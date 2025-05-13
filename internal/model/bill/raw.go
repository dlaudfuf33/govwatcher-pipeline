package bill

import (
	"strconv"
	"strings"
	"time"

	"gwatch-data-pipeline/internal/logging"
)

type BillRaw struct {
	BillID          string `json:"BILL_ID"`
	BillNo          string `json:"BILL_NO"`
	Title           string `json:"BILL_NAME"`
	Committee       string `json:"COMMITTEE"`
	ProposeDate     string `json:"PROPOSE_DT"`
	ProcResult      string `json:"PROC_RESULT"`
	Age             string `json:"AGE"`
	DetailLink      string `json:"DETAIL_LINK"`
	Proposer        string `json:"PROPOSER"`
	MemberListURL   string `json:"MEMBER_LIST"`
	LawProcDate     string `json:"LAW_PROC_DT"`
	LawPresentDate  string `json:"LAW_PRESENT_DT"`
	LawSubmitDate   string `json:"LAW_SUBMIT_DT"`
	CmtProcResultCd string `json:"CMT_PROC_RESULT_CD"`
	CmtProcDate     string `json:"CMT_PROC_DT"`
	CmtPresentDate  string `json:"CMT_PRESENT_DT"`
	CommitteeDate   string `json:"COMMITTEE_DT"`
	ProcDate        string `json:"PROC_DT"`
	CommitteeID     string `json:"COMMITTEE_ID"`
	PubProposer     string `json:"PUBL_PROPOSER"`
	RstProposer     string `json:"RST_PROPOSER"`
	LawProcResultCd string `json:"LAW_PROC_RESULT_CD"`
}

func (r BillRaw) ToEntity(summary string, currentStep string, committeeID uint64) Bill {
	proposeDate := parseDate(r.ProposeDate)
	lawProcDate := parseDate(r.LawProcDate)
	lawPresentDate := parseDate(r.LawPresentDate)
	lawSubmitDate := parseDate(r.LawSubmitDate)
	cmtProcDate := parseDate(r.CmtProcDate)
	cmtPresentDate := parseDate(r.CmtPresentDate)
	committeeDate := parseDate(r.CommitteeDate)
	procDate := parseDate(r.ProcDate)

	age, err := strconv.Atoi(r.Age)
	if err != nil {
		logging.Warnf("Invalid age for BillID %s: %v", r.BillID, err)
		age = 0
	}

	entity := Bill{
		BillID:          r.BillID,
		BillNo:          r.BillNo,
		Title:           r.Title,
		CommitteeID:     committeeID,
		Age:             age,
		ProposeDate:     proposeDate,
		LawProcDate:     lawProcDate,
		LawPresentDate:  lawPresentDate,
		LawSubmitDate:   lawSubmitDate,
		CmtProcDate:     cmtProcDate,
		CmtPresentDate:  cmtPresentDate,
		CommitteeDate:   committeeDate,
		ProcDate:        procDate,
		Result:          r.ProcResult,
		LawProcResultCd: r.LawProcResultCd,
		CmtProcResultCd: r.CmtProcResultCd,
		DetailLink:      r.DetailLink,
		Summary:         summary,
		CurrentStep:     currentStep,
	}
	return entity
}

func parseDate(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil
	}
	return &t
}
