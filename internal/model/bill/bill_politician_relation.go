package bill

import (
	"time"
)

type BillPoliticianRelation struct {
	ID            uint64 `gorm:"primaryKey"`
	BillID        string `gorm:"index"`      // bills.BillID 참조
	PoliticianID  uint64 `gorm:"index"`      // politicians.ID 참조
	Role          string                     // "대표발의" or "공동발의"
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
