FROM --platform=$BUILDPLATFORM golang:1.25-alpine@sha256:06cdd34bd531b810650e47762c01e025eb9b1c7eadd191553b91c9f2d549fae8 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -extldflags -static" -o /out/shadow-empire-bot .

FROM gcr.io/distroless/static:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0
WORKDIR /app
COPY --from=builder /out/shadow-empire-bot /app/shadow-empire-bot
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data
USER nonroot:nonroot

ENTRYPOINT ["/app/shadow-empire-bot"]
