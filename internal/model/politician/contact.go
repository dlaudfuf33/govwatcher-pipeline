package politician

import "time"

type PoliticianContact struct {
	PoliticianID uint64 `gorm:"primaryKey"` // 1:1 관계
	Phone        string
	Email        string
	Homepage     string
	OfficeRoom   string
	Staff        string
	Secretary    string
	Secretary2   string
	UpdatedAt  time.Time
}
