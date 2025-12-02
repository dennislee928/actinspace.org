FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY space-soc/backend/ ./space-soc/backend/
COPY go.mod ./

WORKDIR /app/space-soc/backend/cmd/space-soc
RUN CGO_ENABLED=1 GOOS=linux go build -o /space-soc-backend .

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /space-soc-backend .

EXPOSE 8080
CMD ["./space-soc-backend"]


