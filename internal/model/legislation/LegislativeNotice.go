package model

import "time"

type LegislativeNotice struct {
	ID            uint   `gorm:"primaryKey"`
	BillID        string `gorm:"uniqueIndex"`
	StartDate     time.Time
	EndDate       time.Time
	CommentsURL   string
	CommentsCount int
	ViewCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}