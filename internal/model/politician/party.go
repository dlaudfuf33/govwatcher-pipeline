package politician

type Party struct {
	ID          uint64 `gorm:"primaryKey"`
	Name        string `gorm:"unique;not null"`
	Color       string
	LogoURL     string
	Description string
}
