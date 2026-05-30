FROM golang:1.24-alpine AS base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

FROM base AS test
RUN go test ./...

FROM base AS build
RUN mkdir -p /out/data && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/pack-calculator \
    ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out/pack-calculator /app/pack-calculator
COPY --from=build --chown=nonroot:nonroot /out/data /app/data

USER nonroot:nonroot

ENV PORT=8080
ENV PACK_SIZES_FILE=/app/data/pack_sizes.json
ENV LOG_LEVEL=info
ENV SHUTDOWN_TIMEOUT=10s
ENV READ_HEADER_TIMEOUT=5s

EXPOSE 8080

ENTRYPOINT ["/app/pack-calculator"]
