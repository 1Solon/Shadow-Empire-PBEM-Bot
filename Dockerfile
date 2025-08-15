## Build the app
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -extldflags -static" -o /out/shadow-empire-bot .

## Run the app (distroless static, non-root)
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/shadow-empire-bot /app/shadow-empire-bot
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data
USER nonroot:nonroot

ENTRYPOINT ["/app/shadow-empire-bot"]
