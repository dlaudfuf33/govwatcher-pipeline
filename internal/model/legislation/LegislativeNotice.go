package model

import "time"

type LegislativeNotice struct {
	ID           uint64 `gorm:"primaryKey"`
	BillID       uint64 `gorm:"not null;index;column:bill_id"`
	StartDate    *time.Time
	EndDate      *time.Time
	OpinionUrl   string
	OpinionCount int
	Views        int
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}
