package model

import "time"

type LegislativeOpinion struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"`
	BillID      string     `gorm:"not null;"`
	OpnNo       uint64
	Subject     string
	Content     string
	Author      string
	CreatedAt   time.Time
	IsAnonymous *bool
	Agreement   *bool
}