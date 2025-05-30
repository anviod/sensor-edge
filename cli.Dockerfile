# cli.Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY ./cli ./cli
COPY ./core ./core
COPY ./models ./models
COPY ./types ./types
COPY go.mod go.sum ./
RUN cd cli && go build -o /app/sensor-edge ./main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/sensor-edge ./
ENTRYPOINT ["./sensor-edge"]
