package model

import "time"

type LegislativeOpinion struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	NoticeID  uint64 `gorm:"not null;index"` // Fk LegislativeNotice.ID
	OpnNo     uint64 `gorm:"index"`
	Subject   string `gorm:"type:text"`
	Content   string `gorm:"type:text"`
	Author    string
	Agreement string
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
