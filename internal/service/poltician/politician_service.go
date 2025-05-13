package poltician

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"gorm.io/gorm/clause"

	politicianAPI "gwatch-data-pipeline/internal/api/politician"
	"gwatch-data-pipeline/internal/api/repository"
	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/logging"
	"gwatch-data-pipeline/internal/model/politician"
)

// 역대 의원 데이터 수집
func ImportAllPoliticians() {
	apiKey := util.GetNA()
	currentUnit, err := GetCurrentUnitFromAPI(apiKey)
	if err != nil {
		logging.Errorf("failed to get current unit: %v", err)
		return
	}
	ImportHistoricalPoliticians(apiKey, currentUnit)
	ImportCurrentPoliticians(apiKey)
	ImportPoliticianSNS(apiKey)
}

// 현역 국회의원 데이터 갱신
func UpdateCurrentPoliticians() {
	apiKey := util.GetNA()
	ImportCurrentPoliticians(apiKey)
	ImportPoliticianSNS(apiKey)
}

// 역대 국회의원 인적사항 api 호출 및 저장하는 함수
func ImportHistoricalPoliticians(apiKey string, maxUnit int) {
	partyCache := make(map[string]uint64)
	committeeCache := make(map[string]uint64)

	for unit := 1; unit <= maxUnit; unit++ {
		for page := 1; ; page++ {
			rows, err := politicianAPI.FetchHistoricalPoliticians(apiKey, fmt.Sprintf("1000%02d", unit), page, 100)
			if errors.Is(err, util.ErrNoData) {
				logging.Warnf("[Unit %d] Page %d: No data found", unit, page)
				break
			}

			if err != nil {
				logging.Errorf("[Unit %d] API request failed: %v", unit, err)
				break
			}

			logging.Debugf("📦 [Unit %d] Page %d: Received %d records", unit, page, len(rows))

			for _, raw := range rows {
				var partyID uint64
				if id, ok := partyCache[raw.PolyNm]; ok {
					partyID = id
				} else {
					id, err := repository.GetOrCreateParty(db.DB, raw.PolyNm)
					if err != nil {
						logging.Errorf("Failed to lookup party %s: %v", raw.PolyNm, err)
						continue
					}
					partyCache[raw.PolyNm] = id
					partyID = id
				}

				var committeeID uint64
				if id, ok := committeeCache[raw.CmitNm]; ok {
					committeeID = id
				} else {
					id, err := repository.GetOrCreateCommittee(db.DB, raw.CmitNm)
					if err != nil {
						logging.Errorf("Failed to lookup committee %s: %v", raw.CmitNm, err)
						committeeID = 0 // fallback
					}
					committeeCache[raw.CmitNm] = id
					committeeID = id
				}

				p, t, _, _, _ := raw.ToEntities(unit, partyID, committeeID)
				logging.Debugf("👤 Attempting to save: (MonaCD : %s)", p.MonaCD)

				upsertPolitician(&p)

				if p.ID == 0 {
					if err := db.DB.Where("mona_cd = ?", p.MonaCD).First(&p).Error; err != nil {
						logging.Errorf("[Failed to fetch ID] (MonaCD : %s)", p.MonaCD)
						continue
					}
				}
				logging.Debugf("Successfully saved: (%s : %d)", p.MonaCD, p.ID)

				t.PoliticianID = p.ID
				upsertTerm(&t)
			}
		}
	}
}

// 현역 국회의원 인적사항 api 호출 및 저장하는 함수
func ImportCurrentPoliticians(apiKey string) {
	knownCurrentUnit, err := GetCurrentUnitFromAPI(apiKey)
	if err != nil {
		logging.Errorf("%v", err)
	}
	logging.Debugf("start %d", knownCurrentUnit)
	partyCache := make(map[string]uint64)
	committeeCache := make(map[string]uint64)

	for page := 1; ; page++ {
		rows, err := politicianAPI.FetchCurrentPoliticians(apiKey, page, 100)
		if err != nil {
			logging.Errorf("[Current Politicians] API request failed: %v", err)
			break
		}
		if len(rows) == 0 {
			break
		}
		for _, raw := range rows {
			re := regexp.MustCompile(`\d+`)
			units := re.FindAllString(raw.Units, -1)
			unitInt := 0
			if len(units) > 0 {
				unitInt, _ = strconv.Atoi(units[len(units)-1])
			} else {
				unitInt = knownCurrentUnit
				logging.Warnf("fallback unit used for (%s): defaulted to %d", raw.MonaCD, unitInt)
			}
			if unitInt == 0 {
				logging.Warnf("⚠️ Unable to extract valid unit from raw.Units: %s (MonaCD: %s)", raw.Units, raw.MonaCD)
			}

			var partyID uint64
			if id, ok := partyCache[raw.PolyNm]; ok {
				partyID = id
			} else {
				id, err := repository.GetOrCreateParty(db.DB, raw.PolyNm)
				if err != nil {
					logging.Errorf("Failed to lookup party %s: %v", raw.PolyNm, err)
					continue
				}
				partyCache[raw.PolyNm] = id
				partyID = id
			}

			var committeeID uint64
			if id, ok := committeeCache[raw.CmitNm]; ok {
				committeeID = id
			} else {
				id, err := repository.GetOrCreateCommittee(db.DB, raw.CmitNm)
				if err != nil {
					logging.Errorf("Failed to lookup committee %s: %v", raw.CmitNm, err)
					committeeID = 0
				}
				committeeCache[raw.CmitNm] = id
				committeeID = id
			}

			p, t, c, _, b := raw.ToEntities(unitInt, partyID, committeeID)

			upsertPolitician(&p)
			if err := db.DB.Where("mona_cd = ?", p.MonaCD).First(&p).Error; err != nil {
				continue
			}

			t.PoliticianID = p.ID
			upsertTerm(&t)
			c.PoliticianID = p.ID
			b.PoliticianID = p.ID

			upsertContact(&c)
			upsertCareer(&b)
		}
	}
	logging.Infof("end %d", knownCurrentUnit)
}

// 국회의원 SNS api 호출 및 저장하는 함수
func ImportPoliticianSNS(apiKey string) {
	for page := 1; ; page++ {
		snsRows, err := politicianAPI.FetchPoliticianSNS(apiKey, page, 100)
		if err != nil {
			logging.Errorf("[SNS] API request failed: %v", err)
			break
		}
		if len(snsRows) == 0 {
			break
		}

		for _, raw := range snsRows {
			var p politician.Politician
			if err := db.DB.Where("mona_cd = ?", raw.MonaCD).First(&p).Error; err != nil {
				continue
			}
			sns := raw.ToEntity(p.ID)
			upsertSNS(&sns)
		}
	}
}

func GetCurrentUnitFromAPI(apiKey string) (int, error) {
	page := 1
	size := 10
	rows, err := politicianAPI.FetchCurrentPoliticians(apiKey, page, size)
	if err != nil {
		return 0, fmt.Errorf("[Current Politicians] API request failed: %v", err)
	}

	re := regexp.MustCompile(`\d+`)
	maxUnit := 0

	for _, row := range rows {
		matches := re.FindAllString(row.Units, -1)
		for _, m := range matches {
			unitInt, _ := strconv.Atoi(m)
			if unitInt > maxUnit {
				maxUnit = unitInt
			}
		}
	}

	if maxUnit == 0 {
		return 0, fmt.Errorf("failed to extract valid unit")
	}
	return maxUnit, nil
}

func upsertPolitician(p *politician.Politician) {
	db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mona_cd"}},
		UpdateAll: true,
	}).Create(p)
}

func upsertTerm(t *politician.PoliticianTerm) {
	result := db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "politician_id"}, {Name: "unit"}},
		UpdateAll: true,
	}).Create(t)

	if result.Error != nil {
		logging.Errorf("Failed to upsert term for %d (unit %d): %v", t.PoliticianID, t.Unit, result.Error)
	} else if result.RowsAffected == 0 {
		logging.Warnf("No term row affected for %d (unit %d)", t.PoliticianID, t.Unit)
	}
}

func upsertContact(c *politician.PoliticianContact) {
	db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "politician_id"}},
		UpdateAll: true,
	}).Create(c)
}

func upsertCareer(b *politician.PoliticianCareer) {
	db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "politician_id"}},
		UpdateAll: true,
	}).Create(b)
}

func upsertSNS(s *politician.PoliticianSNS) {
	db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "politician_id"}},
		UpdateAll: true,
	}).Create(s)
}
