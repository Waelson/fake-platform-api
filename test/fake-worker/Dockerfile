FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o fake-worker ./test/fake-worker

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/fake-worker .
ENTRYPOINT ["./fake-worker"]
