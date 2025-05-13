package model

type Bill struct {
	ID          uint64 `gorm:"primaryKey"`
	BillID      string `gorm:"column:bill_id"` // 예: PRC_로 시작하는 고유값
	BillNo      string `gorm:"column:bill_no"` // 예: 2209769
	Name        string `gorm:"column:title"`   // 법안 이름
	ProposeDate string `gorm:"column:propose_date"`
}
