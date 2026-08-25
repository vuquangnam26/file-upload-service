FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/file-upload-service ./cmd

FROM alpine:latest

COPY --from=builder /usr/local/bin/file-upload-service /usr/local/bin/file-upload-service

WORKDIR /app

EXPOSE 8080

CMD ["/usr/local/bin/file-upload-service"]