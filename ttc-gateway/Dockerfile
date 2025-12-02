FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ttc-gateway/ ./ttc-gateway/
COPY go.mod ./

WORKDIR /app/ttc-gateway/cmd/ttc-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /ttc-gateway .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /ttc-gateway .

EXPOSE 8081
CMD ["./ttc-gateway"]

