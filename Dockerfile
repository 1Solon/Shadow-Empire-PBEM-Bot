## Build the app for the target platform
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# These are provided automatically by BuildKit/Buildx for each target platform
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build a static binary for the requested target OS/ARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -extldflags -static" -o /out/shadow-empire-bot .

## Run the app (distroless static, non-root)
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/shadow-empire-bot /app/shadow-empire-bot
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data
USER nonroot:nonroot

ENTRYPOINT ["/app/shadow-empire-bot"]
