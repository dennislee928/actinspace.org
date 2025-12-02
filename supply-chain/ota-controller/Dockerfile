FROM golang:1.23-alpine AS builder

# 安裝 CGO 依賴（SQLite 需要）
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 複製 go.mod 和 go.sum
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

WORKDIR /app/supply-chain/ota-controller/cmd/ota-controller
RUN CGO_ENABLED=1 GOOS=linux go build -o /ota-controller .

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /ota-controller .

EXPOSE 8084
CMD ["./ota-controller"]

