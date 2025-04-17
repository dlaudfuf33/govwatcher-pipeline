DROP DATABASE IF EXISTS gwatch;
CREATE DATABASE gwatch;
USE gwatch;

-- ===============================
-- 🎩 국회의원 기본 인적사항 테이블
-- ===============================
CREATE TABLE politicians
(
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY, -- 내부 식별자 (Auto Increment)
    mona_cd    VARCHAR(20) UNIQUE NOT NULL,                -- 국회 고유 코드 (MONA_CD)
    name       VARCHAR(100),                               -- 한글 이름
    hanja_name VARCHAR(100),                               -- 한자 이름
    eng_name   VARCHAR(100),                               -- 영문 이름
    birth_date DATE,                                       -- 생년월일
    gender     ENUM ('남', '여'),                            -- 성별
    updated_at DATETIME  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
CREATE INDEX idx_politician_lookup ON politicians (name, hanja_name);
-- 이름 + 한자명 인덱스

-- ===============================
-- 🗳️ 의원의 대수별 정치 이력
-- ===============================
CREATE TABLE politician_terms
(
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    politician_id  BIGINT UNSIGNED NOT NULL, -- politicians 테이블 참조
    unit           INT             NOT NULL, -- 대수 (예: 21)
    party          VARCHAR(100),             -- 소속 정당
    constituency   VARCHAR(100),             -- 지역구
    reelected      VARCHAR(10),              -- 재선 여부
    job_title      VARCHAR(100),             -- 직책
    committee_main VARCHAR(200),             -- 소속 상임위
    committees     TEXT,                     -- 참여 상임위 목록
    updated_at     DATETIME  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (politician_id) REFERENCES politicians (id) ON DELETE CASCADE,
    UNIQUE KEY uniq_term (politician_id, unit)
);
CREATE INDEX idx_term_lookup ON politician_terms (politician_id, party);

-- ===============================
-- ☎️ 의원 연락처
-- ===============================
CREATE TABLE politician_contacts
(
    politician_id BIGINT UNSIGNED PRIMARY KEY, -- politicians 테이블 참조
    phone         VARCHAR(50),                 -- 전화번호
    email         VARCHAR(100),                -- 이메일
    homepage      VARCHAR(255),                -- 홈페이지 URL
    office_room   VARCHAR(100),                -- 의원회관 주소
    staff         VARCHAR(100),                -- 보좌관
    secretary     VARCHAR(100),                -- 비서관
    secretary2    VARCHAR(100),                -- 비서
    updated_at    DATETIME  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (politician_id) REFERENCES politicians (id) ON DELETE CASCADE
);

-- ===============================
-- 🌐 SNS 정보
-- ===============================
CREATE TABLE politician_sns
(
    politician_id BIGINT UNSIGNED PRIMARY KEY,
    twitter_url   VARCHAR(255),
    facebook_url  VARCHAR(255),
    youtube_url   VARCHAR(255),
    blog_url      VARCHAR(255),
    updated_at    DATETIME  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (politician_id) REFERENCES politicians (id) ON DELETE CASCADE
);

-- ===============================
-- 🧾 의원 약력
-- ===============================
CREATE TABLE politician_careers
(
    politician_id BIGINT UNSIGNED PRIMARY KEY,
    career        TEXT,
    updated_at    DATETIME  DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (politician_id) REFERENCES politicians (id) ON DELETE CASCADE
);

-- ===============================
-- 📜 발의 법률안 기본 정보
-- ===============================
CREATE TABLE bills
(
    id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    bill_id            VARCHAR(100) NOT NULL UNIQUE, -- 고유 법안 ID (예: PRC_XXXXX)
    bill_no            VARCHAR(100),                 -- 의안번호
    name               VARCHAR(500),                 -- 법안명
    committee          VARCHAR(255),                 -- 소관위원회
    committee_id       VARCHAR(100),                 -- 소관위원회 ID
    age                VARCHAR(10),                  -- 대수 (예: 21)
    proposer           TEXT,                         -- 제안자 (원문 전체)
    main_proposer      VARCHAR(255),                 -- 대표발의자
    sub_proposers      TEXT,                         -- 공동발의자
    propose_date       DATE,                         -- 제안일
    law_proc_date      DATE,                         -- 법사위 처리일
    law_present_date   DATE,                         -- 법사위 상정일
    law_submit_date    DATE,                         -- 법사위 회부일
    cmt_proc_date      DATE,                         -- 소관위 처리일
    cmt_present_date   DATE,                         -- 소관위 상정일
    committee_date     DATE,                         -- 소관위 회부일
    proc_date          DATE,                         -- 본회의 의결일
    result             VARCHAR(255),                 -- 본회의 심의결과
    law_proc_result_cd VARCHAR(100),                 -- 법사위 처리결과 코드
    cmt_proc_result_cd VARCHAR(100),                 -- 소관위 처리결과 코드
    detail_link        TEXT,                         -- 상세페이지 링크
    summary            TEXT,                         -- 제안이유 및 주요내용
    step_log           TEXT,                         -- 전체 심사진행 단계 문자열
    current_step       VARCHAR(255),                 -- 현재 단계
    created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_bill_no (bill_no)
);

-- ===============================
-- 📊 법안의 심사진행단계 히스토리
-- ===============================
CREATE TABLE bill_status_flows
(
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    bill_id    VARCHAR(100) NOT NULL, -- bills.bill_id 참조
    step_order INT,                   -- 진행순서
    step_name  VARCHAR(255),          -- 단계명 (예: 접수, 위원회 심사)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    -- ✅ 유일성 보장
    CONSTRAINT uniq_bill_step UNIQUE (bill_id, step_order),

    -- ✅ FK + 성능 인덱스
    INDEX idx_bill_step (bill_id, step_order),
    FOREIGN KEY (bill_id) REFERENCES bills (bill_id) ON DELETE CASCADE
);

-- ===============================
-- 👥 법안-의원 관계 테이블 (제안자)
-- ===============================
CREATE TABLE bill_politician_relations
(
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    bill_id       VARCHAR(100)    NOT NULL, -- bills.bill_id 참조
    politician_id BIGINT UNSIGNED NOT NULL, -- politicians.id 참조
    role          VARCHAR(50),              -- 역할: "대표발의" 또는 "공동발의"
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_bill_politician (bill_id, politician_id),
    FOREIGN KEY (bill_id) REFERENCES bills (bill_id) ON DELETE CASCADE,
    FOREIGN KEY (politician_id) REFERENCES politicians (id) ON DELETE CASCADE
);

-- ===============================
-- ✍️ 입법예고 테이블
-- ===============================
CREATE TABLE legislative_notices
(
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,                     -- 입법예고 고유 ID
    bill_id        VARCHAR(100) UNIQUE NOT NULL,                                   -- 관련 법안 ID (bills 테이블과 연결)
    start_date     DATE,                                                           -- 입법예고 시작일
    end_date       DATE,                                                           -- 입법예고 종료일
    comments_url   TEXT,                                                           -- 의견 목록 URL (예: 입법예고 페이지 링크)
    comments_count INT      DEFAULT 0,                                             -- 입법예고에 달린 의견 수 (기본값 0)
    view_count     INT      DEFAULT 0,                                             -- 입법예고 조회수 (기본값 0)
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,                             -- 생성일
    updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, -- 수정일
    FOREIGN KEY (bill_id) REFERENCES bills (bill_id) ON DELETE CASCADE             -- bills 테이블과 관계
);


-- ===============================
-- 💬 입법예고 의견 테이블
-- ===============================
CREATE TABLE legislative_opinions
(
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,                         -- 의견 고유 ID
    bill_id      VARCHAR(100) NOT NULL,                                       -- legislative_notices 테이블과 연결 (입법예고)
    opn_no       BIGINT,
    subject      TEXT,                                                               -- 의견 제목
    content      TEXT,                                                               -- 의견 내용
    author       VARCHAR(100),                                                       -- 의견 작성자
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,                                 -- 의견 작성일
    is_anonymous BOOLEAN  DEFAULT FALSE,                                             -- 비공개 여부 (비공개 시 TRUE)
    agreement    BOOLEAN  DEFAULT NULL,                                              -- 찬성/반대 여부 (NULL: 무응답, TRUE: 찬성, FALSE: 반대)
    UNIQUE KEY uniq_bill_opn (bill_id, opn_no),
    FOREIGN KEY (bill_id) REFERENCES legislative_notices (bill_id) ON DELETE CASCADE -- 입법예고 테이블과 관계
);
