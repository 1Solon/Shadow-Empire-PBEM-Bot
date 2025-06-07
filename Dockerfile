## Build the app
FROM golang:1.24-alpine@sha256:68932fa6d4d4059845c8f40ad7e654e626f3ebd3706eef7846f319293ab5cb7a AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o shadow-empire-bot .

## Run the app
FROM alpine:latest@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c
WORKDIR /app
COPY --from=builder /app/shadow-empire-bot .
RUN mkdir -p /app/data
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data

CMD ["./shadow-empire-bot"]
