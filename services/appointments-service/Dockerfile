FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o appointments-service ./cmd/api

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/appointments-service ./appointments-service

EXPOSE 8082

CMD ["./appointments-service"]
