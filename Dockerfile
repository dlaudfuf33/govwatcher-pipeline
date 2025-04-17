# Dockerfile for govwatch CLI container image
# Stage 1: Build
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# govwatch CLI 앱 빌드 (cmd/govwatch/main.go 기준)
RUN CGO_ENABLED=0 GOOS=linux go build -o govwatch ./cmd/govwatch

# Stage 2: Final
FROM debian:bookworm-slim

WORKDIR /app

# 실행에 필요한 파일만 복사
COPY --from=builder /app/govwatch .

# 실행 시 .env 가 환경에 있다고 가정
CMD ["./govwatch"]
