## Build the app
FROM golang:1.24-alpine@sha256:b4f875e650466fa0fe62c6fd3f02517a392123eea85f1d7e69d85f780e4db1c1 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o shadow-empire-bot .

## Run the app
FROM alpine:latest@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
WORKDIR /app
COPY --from=builder /app/shadow-empire-bot .
RUN mkdir -p /app/data
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data

CMD ["./shadow-empire-bot"]
