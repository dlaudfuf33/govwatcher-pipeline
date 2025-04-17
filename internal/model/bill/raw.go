package bill

import (
	"strings"
	"time"
)

type BillRaw struct {
	BillID           string `json:"BILL_ID"`
	BillNo           string `json:"BILL_NO"`
	BillName         string `json:"BILL_NAME"`
	Committee        string `json:"COMMITTEE"`
	ProposeDate      string `json:"PROPOSE_DT"`
	ProcResult       string `json:"PROC_RESULT"`
	Age              string `json:"AGE"`
	DetailLink       string `json:"DETAIL_LINK"`
	Proposer         string `json:"PROPOSER"`
	MemberListURL    string `json:"MEMBER_LIST"`
	LawProcDate      string `json:"LAW_PROC_DT"`
	LawPresentDate   string `json:"LAW_PRESENT_DT"`
	LawSubmitDate    string `json:"LAW_SUBMIT_DT"`
	CmtProcResultCd  string `json:"CMT_PROC_RESULT_CD"`
	CmtProcDate      string `json:"CMT_PROC_DT"`
	CmtPresentDate   string `json:"CMT_PRESENT_DT"`
	CommitteeDate    string `json:"COMMITTEE_DT"`
	ProcDate         string `json:"PROC_DT"`
	CommitteeID      string `json:"COMMITTEE_ID"`
	PubProposer      string `json:"PUBL_PROPOSER"`
	RstProposer      string `json:"RST_PROPOSER"`
	LawProcResultCd  string `json:"LAW_PROC_RESULT_CD"`
}

func (r BillRaw) ToEntity(summary string, statusSteps []string, currentStep string) (Bill, []BillStatusFlow) {
	proposeDate := parseDate(r.ProposeDate)
	lawProcDate := parseDate(r.LawProcDate)
	lawPresentDate := parseDate(r.LawPresentDate)
	lawSubmitDate := parseDate(r.LawSubmitDate)
	cmtProcDate := parseDate(r.CmtProcDate)
	cmtPresentDate := parseDate(r.CmtPresentDate)
	committeeDate := parseDate(r.CommitteeDate)
	procDate := parseDate(r.ProcDate)

	entity := Bill{
		BillID:          r.BillID,
		BillNo:          r.BillNo,
		Name:            r.BillName,
		Committee:       r.Committee,
		CommitteeID:     r.CommitteeID,
		Age:             r.Age,
		Proposer:        r.Proposer,
		MainProposer:    r.RstProposer,
		SubProposers:    r.PubProposer,
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

	var steps []BillStatusFlow
	for idx, step := range statusSteps {
		if step == "" {
			continue
		}
		steps = append(steps, BillStatusFlow{
			BillID:    r.BillID,
			StepName:  step,       // Step → StepName
			StepOrder: idx + 1,    // Order → StepOrder
		})
	}

	return entity, steps
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
