FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go test ./... && CGO_ENABLED=0 GOOS=linux go build -o /pack-calculator ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=build /pack-calculator /app/pack-calculator

ENV PORT=8080
ENV PACK_SIZES_FILE=/app/data/pack_sizes.json

EXPOSE 8080

CMD ["/app/pack-calculator"]
