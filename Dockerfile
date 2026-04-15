FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrator ./cmd/migrator

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/migrator /app/migrator
COPY ./migrations /app/migrations

ENTRYPOINT ["/app/migrator"]