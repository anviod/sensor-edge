# backend.Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download && go build -o sensor-edge-server ./cmd/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/sensor-edge-server ./
COPY ./config ./config
COPY ./docs ./docs
EXPOSE 8080
CMD ["./sensor-edge-server"]
