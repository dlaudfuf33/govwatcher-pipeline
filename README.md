# 🏛️ GovWatcher Pipeline

`govwatcher-pipeline`은 대한민국 국회의 입법예고, 법안, 국회의원 관련 데이터를 **자동 수집 및 저장**하는 Go 기반 CLI 데이터 파이프라인입니다. CLI 명령어를 통해 다양한 수집 작업을 자동화하고 병렬로 처리합니다.

---

## ✅ 주요 기능

- 국회의원 전체 / 현역 / SNS 정보 수집
- 법률안 전체 수집 및 의원 매핑 (발의자 포함)
- 입법예고 기간 수집 및 의견 엑셀 다운로드
- 의견 본문 크롤링 (공개 의견)
- 병렬 처리 기반의 성능 최적화

---

## 🗂️ 프로젝트 구조 및 설명

```
├── cmd/                              # CLI 명령어 정의 (cobra 기반)
│   ├── govwatch/main.go              # CLI 실행 진입점
│   ├── init.go                       # 초기 전체 수집 (모든 politician, bill, notice, opinion)
│   ├── root.go                       # 루트 명령어 정의
│   ├── update.go                     # 전체 업데이트 (현역 갱신 포함)
│   ├── update1d.go                   # 1일 이내 마감 입법예고 의견만 수집
│   ├── update3d.go                   # 3일 이내 마감 입법예고 의견만 수집
│   └── update7d.go                   # 7일 이내 마감 입법예고 의견만 수집
├── downloads/                        # 수집된 엑셀 파일 저장 위치
│   ├── notice/                       # 입법예고 목록 XLSX
│   └── opinion/                      # 입법예고 의견 XLSX
├── internal/
│   ├── api/                          # Open API 호출 모듈
│   │   ├── bill/
│   │   │   ├── Information.go        # 법안 메타정보 수집 (상세)
│   │   │   ├── bill_detail.go        # 법안 상세페이지 크롤링
│   │   │   ├── bill_list.go          # 법안 목록 API 수집
│   │   │   └── bill_proposer.go      # 법안 발의자 목록 크롤링
│   │   ├── legislation/
│   │   │   ├── contents.go           # 의견 본문 크롤링 (FetchOpinionContent)
│   │   │   ├── download.go           # 엑셀 다운로드 POST 요청 생성
│   │   │   ├── list.go               # 입법예고 XML 목록 API 파싱
│   │   │   ├── listcrawler.go        # 입법예고 리스트 HTML 크롤링
│   │   │   ├── opinion.go            # JSON API로 의견 목록 조회
│   │   │   └── session.go            # chromedp 세션 초기화 및 쿠키/토큰 추출
│   │   ├── politician/
│   │   │   ├── politician_all.go     # 역대 의원 전체 목록 API
│   │   │   ├── politician_current.go # 현역 의원 API
│   │   │   ├── politician_history.go # 과거 의원 정보
│   │   │   └── politician_sns.go     # SNS 정보 수집
│   │   └── util/
│   │       ├── constant.go           # 상수 정의
│   │       └── util.go               # 공통 함수 (MakeRequest 등)
│   ├── db/
│   │   └── mysql.go                  # MySQL DB 연결 및 초기화
│   ├── logging/
│   │   └── logging.go                # 로그 출력 설정
│   ├── model/                        # DB 저장용 구조체 (GORM)
│   │   ├── SessionInfo.go            # chromedp 세션 정보를 담는 구조체
│   │   ├── bill/
│   │   │   ├── bill.go               # 법안 기본 정보
│   │   │   ├── bill_politician_relation.go  # 법안-의원 관계 (발의자, 공동발의자)
│   │   │   ├── bill_status_flow.go  # 심사진행 단계
│   │   │   └── raw.go               # 법안 API → Entity 변환
│   │   ├── legislation/
│   │   │   ├── LegislativeNotice.go # 입법예고 기간 및 메타 정보
│   │   │   ├── LegislativeOpinion.go # 입법예고 의견 정보
│   │   │   └── bill.go              # 입법예고와 연계된 법안 정보
│   │   └── politician/
│   │       ├── base.go              # 국회의원 공통 인적사항
│   │       ├── career.go            # 경력
│   │       ├── contact.go           # 연락처
│   │       ├── raw.go               # API → 구조체 변환
│   │       ├── sns.go               # SNS 정보
│   │       ├── sns_raw.go           # SNS API 변환용
│   │       └── term.go              # 대수별 의원 정보
│   └── service/                     # 실제 로직 실행 모듈
│       ├── bill/
│       │   └── bill_service.go      # 법안 전체 수집 및 DB 저장
│       ├── legislation/
│       │   ├── notice_service.go    # 입법예고 목록 및 기간 수집
│       │   └── opinion_service.go   # 의견 다운로드 및 파싱
│       └── poltician/
│           └── politician_service.go # 국회의원 정보 수집 및 분리 저장
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 실행 방법

```bash
# 환경변수 설정 (.env 또는 export)
export DB_USER=root
export DB_PASS=yourpassword
export DB_HOST=localhost
export DB_PORT=3306
export NA_KEY=공공데이터_API키
export LOG_LEVEL=INFO

# 전체 초기 수집
go run cmd/govwatch/main.go init

# 업데이트 (입법예고 + 법안 + 현역 의원)
go run cmd/govwatch/main.go update

# 마감 임박 의견만 수집 (1~7일 단위 선택)
go run cmd/govwatch/main.go update1d
go run cmd/govwatch/main.go update3d
go run cmd/govwatch/main.go update7d
```


