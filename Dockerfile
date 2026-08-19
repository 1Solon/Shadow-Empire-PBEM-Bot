FROM --platform=$BUILDPLATFORM golang:1.25-alpine@sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -extldflags -static" -o /out/shadow-empire-bot .

FROM gcr.io/distroless/static:nonroot@sha256:2b7c93f6d6648c11f0e80a48558c8f77885eb0445213b8e69a6a0d7c89fc6ae4
WORKDIR /app
COPY --from=builder /out/shadow-empire-bot /app/shadow-empire-bot
VOLUME /app/data
ENV WATCH_DIRECTORY=/app/data
USER nonroot:nonroot

ENTRYPOINT ["/app/shadow-empire-bot"]
