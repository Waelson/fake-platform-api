FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o fake-platform-api ./cmd/fake-platform-api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/fake-platform-api .
EXPOSE 8080
CMD ["./fake-platform-api"]
