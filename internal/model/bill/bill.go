package bill

import "time"

type Bill struct {
	ID               uint64     `gorm:"primaryKey"`
	BillID           string     `gorm:"uniqueIndex"` // PRC_ 로 시작하는 고유 의안 ID
	BillNo           string     // 의안번호
	Name             string     // 법안명
	Committee        string     // 소관위원회
	CommitteeID      string     // 소관위원회 ID
	Age              string     // 대수 (21대, 20대...)
	Proposer         string     // 원문상 제안자 문자열
	MainProposer     string     // 대표발의자
	SubProposers     string     // 공동발의자 목록 (문자열)
	ProposeDate      *time.Time // 제안일
	LawProcDate      *time.Time // 법사위 처리일
	LawPresentDate   *time.Time // 법사위 상정일
	LawSubmitDate    *time.Time // 법사위 회부일
	CmtProcDate      *time.Time // 소관위 처리일
	CmtPresentDate   *time.Time // 소관위 상정일
	CommitteeDate    *time.Time // 소관위 회부일
	ProcDate         *time.Time // 의결일
	Result           string     // 본회의 심의결과
	LawProcResultCd  string     // 법사위 처리결과 코드
	CmtProcResultCd  string     // 소관위 처리결과 코드
	DetailLink       string     // 상세페이지 링크
	Summary          string     // 제안 이유 및 주요내용
	StepLog          string     // 심사진행 전체 단계 문자열
	CurrentStep      string     // 현재 심사진행 단계
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
