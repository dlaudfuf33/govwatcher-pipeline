package politician

import "time"

type Politician struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	MonaCD     string     `gorm:"unique;not null"` // 국회의원 고유 코드
	Name       string
	HanjaName  string
	EngName    string
	BirthDate  *time.Time
	Gender     string // '남' or '여'
	UpdatedAt  time.Time
}
