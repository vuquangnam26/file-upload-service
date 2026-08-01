FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY . ./
RUN go build -o /usr/local/bin/file-upload-service ./cmd/api

FROM alpine:latest
COPY --from=builder /usr/local/bin/file-upload-service /usr/local/bin/file-upload-service
WORKDIR /app
EXPOSE 8080
CMD ["/usr/local/bin/file-upload-service"]
