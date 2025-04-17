package politician

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PoliticianRaw: 국회 API에서 직접 받아오는 필드 구조
type PoliticianRaw struct {
	MonaCD       string `json:"MONA_CD"`
	HgNm         string `json:"HG_NM"`
	HjNm         string `json:"HJ_NM"`
	EngNm        string `json:"ENG_NM"`
	BthDate      string `json:"BTH_DATE"`
	SexGbnNm     string `json:"SEX_GBN_NM"`

	PolyNm       string `json:"POLY_NM"`
	OrigNm       string `json:"ORIG_NM"`
	ReeleGbnNm   string `json:"REELE_GBN_NM"`
	Units        string `json:"UNITS"`
	ElectGbnNm   string `json:"ELECT_GBN_NM"`

	JobResNm     string `json:"JOB_RES_NM"`
	CmitNm       string `json:"CMIT_NM"`
	Cmits        string `json:"CMITS"`

	TelNo        string `json:"TEL_NO"`
	Email        string `json:"E_MAIL"`
	Homepage     string `json:"HOMEPAGE"`
	Staff        string `json:"STAFF"`
	Secretary    string `json:"SECRETARY"`
	Secretary2   string `json:"SECRETARY2"`
	AssemAddr    string `json:"ASSEM_ADDR"`

	MemTitle     string `json:"MEM_TITLE"`

	// SNS는 후처리 시에 채워질 수도 있음
	TwitterURL   string
	FacebookURL  string
	YoutubeURL   string
	BlogURL      string
}

// ToEntities: PoliticianRaw → 분리된 5개 구조체로 변환
func (r PoliticianRaw) ToEntities(unit int) (
	Politician,
	PoliticianTerm,
	PoliticianContact,
	PoliticianSNS,
	PoliticianCareer,
) {
	birthDate := parseBirthDate(r.BthDate)

	p := Politician{
		MonaCD:     r.MonaCD,
		Name:       r.HgNm,
		HanjaName:  r.HjNm,
		EngName:    r.EngNm,
		BirthDate:  birthDate,
		Gender:     r.SexGbnNm,
	}

	t := PoliticianTerm{
		Unit:           unit,
		Party:          r.PolyNm,
		Constituency:   r.OrigNm,
		Reelected:      r.ReeleGbnNm,
		JobTitle:       r.JobResNm,
		CommitteeMain:  r.CmitNm,
		Committees:     r.Cmits,
	}

	c := PoliticianContact{
		Phone:      r.TelNo,
		Email:      r.Email,
		Homepage:   r.Homepage,
		OfficeRoom: r.AssemAddr,
		Staff:      r.Staff,
		Secretary:  r.Secretary,
		Secretary2: r.Secretary2,
	}

	s := PoliticianSNS{
		TwitterURL:  r.TwitterURL,
		FacebookURL: r.FacebookURL,
		YoutubeURL:  r.YoutubeURL,
		BlogURL:     r.BlogURL,
	}

	b := PoliticianCareer{
		Career: r.MemTitle,
	}

	return p, t, c, s, b
}

func parseBirthDate(raw string) *time.Time {
	if raw == "" {
		return nil
	}

	// 1. 정상 포맷 시도
	layouts := []string{"2006-01-02", "20060102"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			if t.After(time.Now()) {
				// 미래 날짜 → 1919-03-01로 치환
				return ptr(time.Date(1919, 3, 1, 0, 0, 0, 0, time.UTC))
			}
			return &t
		}
	}

	// 2. 년, 월, 일 분해
	var year, month, day string
	if strings.Contains(raw, "-") {
		parts := strings.Split(raw, "-")
		if len(parts) > 0 {
			year = parts[0]
		}
		if len(parts) > 1 {
			month = parts[1]
		}
		if len(parts) > 2 {
			day = parts[2]
		}
	} else if len(raw) == 8 {
		year, month, day = raw[:4], raw[4:6], raw[6:8]
	} else if len(raw) == 4 {
		year = raw
	}

	// 3. 보정: 잘못된 값 대체
	if year != "" {
		if month == "00" || month == "" || month == "-" {
			month = "01"
		}
		if day == "00" || day == "" || day == "-" {
			day = "01"
		}
		dateStr := fmt.Sprintf("%s-%s-%s", year, month, day)
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			if t.After(time.Now()) {
				return ptr(time.Date(1919, 3, 1, 0, 0, 0, 0, time.UTC))
			}
			return &t
		} else {
			// 파싱 실패 = 존재하지 않는 날짜 → 연도 기준으로 01-01
			if y, err := strconv.Atoi(year); err == nil {
				return ptr(time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC))
			}
		}
	}

	return nil
}

// 헬퍼
func ptr(t time.Time) *time.Time {
	return &t
}
