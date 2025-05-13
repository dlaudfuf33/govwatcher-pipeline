package bill

import (
	"time"
)

type BillPoliticianRelation struct {
	ID           uint64 `gorm:"primaryKey"`
	BillID       uint64 `gorm:"index"`
	PoliticianID uint64 `gorm:"index"` // politicians.ID 참조
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
