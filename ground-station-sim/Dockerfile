FROM golang:1.22-alpine AS builder

WORKDIR /app

# 複製 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

WORKDIR /app/ground-station-sim/cmd/ground-station-sim
RUN CGO_ENABLED=0 GOOS=linux go build -o /ground-station-sim .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /ground-station-sim .

CMD ["./ground-station-sim"]
