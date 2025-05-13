package bill

import "time"

type BillStatusFlow struct {
	ID        uint64 `gorm:"primaryKey"`
	BillID    uint64 `gorm:"index"`
	StepOrder int    // 순서 (0부터 시작)
	StepName  string // 단계 이름 (예: "접수", "위원회 심사", "임기만료폐기")
	CreatedAt time.Time
	UpdatedAt time.Time
}
