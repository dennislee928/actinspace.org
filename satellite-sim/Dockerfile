FROM golang:1.23-alpine AS builder

WORKDIR /app

# 複製 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

RUN go build -o /satellite-sim ./satellite-sim/cmd/satellite-sim

FROM alpine:3.20

WORKDIR /app

# 複製可執行檔並設置執行權限
COPY --from=builder /satellite-sim /app/satellite-sim
RUN chmod +x /app/satellite-sim

ENV PORT=8082

EXPOSE 8082

ENTRYPOINT ["/app/satellite-sim"]


