FROM golang:1.22-alpine AS builder

WORKDIR /app

# 複製 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

WORKDIR /app/ttc-gateway/cmd/ttc-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /ttc-gateway .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /ttc-gateway .

EXPOSE 8081
CMD ["./ttc-gateway"]
