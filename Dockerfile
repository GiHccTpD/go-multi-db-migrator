FROM golang:1.23 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/go-multi-db-migrator ./cmd/migrator

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/go-multi-db-migrator /app/go-multi-db-migrator
COPY ./migrations /app/migrations

ENTRYPOINT ["/app/go-multi-db-migrator"]