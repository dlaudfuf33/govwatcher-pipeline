package politician

import "time"

type PoliticianSNS struct {
	PoliticianID uint64 `gorm:"primaryKey"`
	TwitterURL   string
	FacebookURL  string
	YoutubeURL   string
	BlogURL      string
	UpdatedAt  time.Time
}
