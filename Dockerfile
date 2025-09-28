FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# APIサーバ
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
# バッチ（ingest）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ingest ./cmd/ingest

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/ingest .
EXPOSE 8080
CMD ["./server"]