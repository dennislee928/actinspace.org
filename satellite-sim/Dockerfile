FROM golang:1.22-alpine AS builder

WORKDIR /app

# 複製 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./
RUN go mod download

# 複製所有原始碼
COPY . .

RUN go build -o satellite-sim ./satellite-sim/cmd/satellite-sim

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/satellite-sim /app/satellite-sim

ENV PORT=8082

EXPOSE 8082

ENTRYPOINT ["/app/satellite-sim"]


