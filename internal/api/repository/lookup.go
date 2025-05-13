package repository

import (
	"errors"

	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/model/politician"
)

// GetOrCreateParty: 이름으로 조회, 없으면 insert
func GetOrCreateParty(db *gorm.DB, name string) (uint64, error) {
    var party politician.Party
    err := db.First(&party, "name = ?", name).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        party = politician.Party{Name: name}
        if err := db.Create(&party).Error; err != nil {
            return 0, err
        }
        return party.ID, nil
    } else if err != nil {
        return 0, err
    }
    return party.ID, nil
}

func GetOrCreateCommittee(db *gorm.DB, name string) (uint64, error) {
    var committee politician.Committee
    err := db.First(&committee, "name = ?", name).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        committee = politician.Committee{Name: name}
        if err := db.Create(&committee).Error; err != nil {
            return 0, err
        }
        return committee.ID, nil
    } else if err != nil {
        return 0, err
    }
    return committee.ID, nil
}
