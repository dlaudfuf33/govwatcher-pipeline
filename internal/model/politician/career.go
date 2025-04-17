package politician

import "time"

type PoliticianCareer struct {
	PoliticianID uint64 `gorm:"primaryKey"`
	Career       string
	UpdatedAt  time.Time
}
