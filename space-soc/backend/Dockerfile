FROM golang:1.23-alpine AS builder

# 安裝 CGO 依賴（SQLite 需要）
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 複製 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

WORKDIR /app/space-soc/backend/cmd/space-soc
RUN CGO_ENABLED=1 GOOS=linux go build -o /space-soc-backend .

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /space-soc-backend .

EXPOSE 8080
CMD ["./space-soc-backend"]