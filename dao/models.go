package dao

import "gorm.io/gorm"

type Block struct {
	gorm.Model

	LastHeight uint64
	LastHash   string `gorm:"type:char(66)"`
}

func NewBlock() *Block {
	return &Block{}
}

func (t *Block) Save(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Block{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return tx.Create(&t).Error
	}
	return tx.Updates(&t).Error
}

func (t *Block) Get() (*Block, error) {
	var result Block
	if err := DB().Model(&Block{}).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

type Address struct {
	gorm.Model

	Parent     string `gorm:"type:varchar(42)"`
	Address    string `gorm:"type:varchar(42)"`
	Generation uint
}

func (t *Address) Save(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Address{}).Where("`address` = ?", t.Address).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&t).Error
}

func (t *Address) FindByAddress(tx *gorm.DB) (*Address, error) {
	var result Address
	if err := tx.Model(&Address{}).Where("`address` = ?", t.Address).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}
