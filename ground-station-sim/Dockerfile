FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ground-station-sim/ ./ground-station-sim/
COPY go.mod ./

WORKDIR /app/ground-station-sim/cmd/ground-station-sim
RUN CGO_ENABLED=0 GOOS=linux go build -o /ground-station-sim .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /ground-station-sim .

CMD ["./ground-station-sim"]

