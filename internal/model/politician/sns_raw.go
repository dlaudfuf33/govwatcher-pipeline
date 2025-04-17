package politician

// PoliticianSNSRaw: SNS API에서 수신되는 데이터 형식
type PoliticianSNSRaw struct {
	Name       string `json:"HG_NM"`
	MonaCD     string `json:"MONA_CD"`
	TwitterURL string `json:"T_URL"`
	FacebookURL string `json:"F_URL"`
	YoutubeURL string `json:"Y_URL"`
	BlogURL     string `json:"B_URL"`
}

// ToEntity: DB 저장용 PoliticianSNS 구조체로 변환
func (r PoliticianSNSRaw) ToEntity(politicianID uint64) PoliticianSNS {
	return PoliticianSNS{
		PoliticianID: politicianID,
		TwitterURL:   r.TwitterURL,
		FacebookURL:  r.FacebookURL,
		YoutubeURL:   r.YoutubeURL,
		BlogURL:      r.BlogURL,
	}
}
