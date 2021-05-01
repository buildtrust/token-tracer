package services

import (
	"context"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/buildtrust/token-tracer/config"
	"github.com/buildtrust/token-tracer/dao"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Tip struct {
	Height uint64
	Hash   string
}

type Tracer struct {
	lastTip  *Tip
	client   *ethclient.Client
	tipChan  chan *Tip
	contract common.Address
}

func NewTracer() (*Tracer, error) {
	client, err := ethclient.Dial(config.GetConfig().RPC)
	if err != nil {
		return nil, err
	}
	last, err := dao.NewBlock().Get()
	if err != nil {
		return nil, err
	}
	if last == nil {
		header, err := client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(config.GetConfig().StartBlock))
		if err != nil {
			return nil, err
		}
		last = &dao.Block{
			LastHeight: header.Number.Uint64(),
			LastHash:   header.Hash().String(),
		}
		if err := last.Save(dao.DB()); err != nil {
			return nil, err
		}
	}
	return &Tracer{
		lastTip: &Tip{
			Height: last.LastHeight,
			Hash:   last.LastHash,
		},
		client:   client,
		tipChan:  make(chan *Tip),
		contract: common.HexToAddress(config.GetConfig().Contract),
	}, nil
}

func (t *Tracer) Trace() {
	go func() {
		for {
			tip := <-t.tipChan
			if t.lastTip.Height == tip.Height {
				continue
			}
			log.Printf("trace log: %d -> %d\n", t.lastTip.Height, tip.Height)
			query := ethereum.FilterQuery{
				FromBlock: new(big.Int).SetUint64(t.lastTip.Height),
				ToBlock:   new(big.Int).SetUint64(tip.Height),
				Addresses: []common.Address{t.contract},
			}
			logs, err := t.client.FilterLogs(context.Background(), query)
			if err != nil {
				log.Fatalf("can't filter logs: %v", err)
			}
			for _, l := range logs {
				// Transfer
				if l.Topics[0].String() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
					from := strings.ToLower(common.BytesToAddress(common.TrimLeftZeroes(l.Topics[1][:])).String())
					to := strings.ToLower(common.BytesToAddress(common.TrimLeftZeroes(l.Topics[2][:])).String())
					data := new(big.Int).SetBytes(l.Data)
					trace := false
					if config.GetConfig().ContainAddress(from) {
						trace = true
						addr := dao.Address{
							Parent:     from,
							Address:    to,
							Generation: 1,
						}
						if err := addr.Save(dao.DB()); err != nil {
							log.Fatalf("save address error: %v", err)
						}
					}
					if !trace {
						addr, err := dao.NewAddress().FindByAddress(from)
						if err != nil {
							log.Fatalf("query address error: %v", err)
						}
						if addr != nil && addr.Generation < 3 {
							trace = true
							addr := dao.Address{
								Parent:     addr.Address,
								Address:    to,
								Generation: addr.Generation + 1,
							}
							if err := addr.Save(dao.DB()); err != nil {
								log.Fatalf("save address error: %v", err)
							}
						}
					}
					if trace {
						transfer := &dao.Transfer{
							Height: l.BlockNumber,
							Hash:   l.TxHash.String(),
							From:   from,
							To:     to,
							Amount: data.String(),
						}
						if err := transfer.Save(dao.DB()); err != nil {
							log.Fatalf("save transfer error: %v", err)
						}
					}
				}
			}
			t.lastTip = &Tip{
				Height: tip.Height,
				Hash:   tip.Hash,
			}
			block := &dao.Block{
				LastHeight: tip.Height,
				LastHash:   tip.Hash,
			}
			if err := block.Save(dao.DB()); err != nil {
				log.Fatalf("save block error: %v", err)
			}
		}
	}()
	go func() {
		blockNumber, err := t.client.BlockNumber(context.Background())
		if err != nil {
			log.Fatalf("can't get block number: %v", err)
		}
		for {
			nextHeight := blockNumber
			if blockNumber-t.lastTip.Height > 1000 {
				nextHeight = t.lastTip.Height + 1000
			}
			nextHeader, err := t.client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(config.GetConfig().StartBlock))
			if err != nil {
				log.Fatalf("can't get block header: %v", err)
			}
			t.tipChan <- &Tip{
				Height: nextHeight,
				Hash:   nextHeader.Hash().String(),
			}
			if nextHeight == blockNumber {
				break
			}
		}
		ticker := time.NewTicker(30 * time.Second)
		go func() {
			for {
				<-ticker.C
				if blockNumber, err := t.client.BlockNumber(context.Background()); err != nil {
					log.Fatalf("can't get block number: %v", err)
				} else {
					header, err := t.client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(config.GetConfig().StartBlock))
					if err != nil {
						log.Fatalf("can't get block header: %v", err)
					}
					t.tipChan <- &Tip{
						Height: blockNumber,
						Hash:   header.Hash().String(),
					}
				}
			}
		}()
	}()
}
