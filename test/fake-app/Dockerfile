FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o fake-app ./test/fake-app

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/fake-app .
EXPOSE 3000
ENTRYPOINT ["./fake-app"]
