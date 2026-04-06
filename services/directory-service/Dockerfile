FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o directory-service ./cmd/api

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/directory-service ./directory-service

EXPOSE 8081

CMD ["./directory-service"]
