package politician

import "time"

type PoliticianTerm struct {
	ID             uint64 `gorm:"primaryKey"`
	PoliticianID   uint64 `gorm:"index"`  // 외래키
	Unit           int    // 제XX대
	Party          string
	Constituency   string
	Reelected      string
	JobTitle       string
	CommitteeMain  string
	Committees     string
	UpdatedAt      time.Time
}

