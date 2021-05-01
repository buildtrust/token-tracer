package services

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
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
				fmt.Println(l.Topics[0].String())
				// Transfer
				if l.Topics[0].String() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
					from := "0x" + hex.EncodeToString(l.Topics[1][:])
					fmt.Println(from)
				}
			}
			t.lastTip = &Tip{
				Height: tip.Height,
				Hash:   tip.Hash,
			}
			// TODO save last
		}
	}()
	go func() {
		blockNumber, err := t.client.BlockNumber(context.Background())
		if err != nil {
			log.Fatalf("can't get block number: %v", err)
		}
		for {
			nextHeight := blockNumber
			if blockNumber-t.lastTip.Height > 100 {
				nextHeight = t.lastTip.Height + 100
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
