FROM golang:1.26 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o bcv ./cmd/book-club-vote/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bcv /usr/local/bin/bcv
EXPOSE 23234
ENTRYPOINT ["/usr/local/bin/bcv"]
