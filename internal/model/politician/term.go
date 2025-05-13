package politician

import "time"

type PoliticianTerm struct {
	ID           uint64 `gorm:"primaryKey"`
	PoliticianID uint64 `gorm:"index"`
	Unit         int
	PartyID      uint64
	Constituency string
	Reelected    string
	JobTitle     string
	CommitteeID  uint64
	UpdatedAt    time.Time
}
