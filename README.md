# Order Packs Calculator

A Go HTTP application that calculates which complete packs should be shipped for a customer order.

The calculation follows the rules from the challenge in priority order:

1. Only whole packs can be shipped.
2. Ship the fewest total items that still fulfils the order.
3. If multiple allocations ship that same total, use the fewest packs.

Pack sizes are configurable at runtime through the API or UI and are persisted to a JSON file.

## Features

- Go HTTP API
- Browser UI for changing pack sizes and calculating orders
- Runtime configurable pack sizes
- JSON persistence for pack-size configuration
- Structured logs with `zerolog`
- Graceful shutdown on `SIGINT` and `SIGTERM`
- Unit and HTTP API tests
- Dockerfile for containerized deployment

## Architecture

The project is organized with DDD-style boundaries:

- `internal/domain/orderpacks`: core business language and rules. It owns pack-size validation, normalization, and the shipment-planning algorithm.
- `internal/application/orderpacks`: use cases. It coordinates pack-size configuration and shipment calculation through a repository interface.
- `internal/infrastructure/config`: typed environment configuration with defaults and validation.
- `internal/infrastructure/persistence/packsize`: JSON-file persistence adapter for configurable pack sizes.
- `internal/interfaces/httpapi`: HTTP adapter that maps requests/responses to application use cases.
- `cmd/server`: composition root. It wires infrastructure, application services, domain services, HTTP routes, logging, and graceful shutdown.

The domain package does not depend on HTTP or persistence details. The application package depends on a repository interface, while the JSON store implements that interface from the infrastructure layer.

## Run Locally

```bash
go run ./cmd/server
```

Open [http://localhost:8080](http://localhost:8080).

If your local Go environment has module mode disabled, run commands with `GO111MODULE=on`.

Optional environment variables:

- `PORT`: server port, defaults to `8080`
- `PACK_SIZES_FILE`: JSON persistence file, defaults to `data/pack_sizes.json`
- `LOG_LEVEL`: logging level, defaults to `debug`; supported values include `debug`, `info`, `warn`, and `error`
- `SHUTDOWN_TIMEOUT`: graceful shutdown timeout, defaults to `10s`
- `READ_HEADER_TIMEOUT`: HTTP read-header timeout, defaults to `5s`

Use `.env.example` as the deployment/configuration template. The app reads environment variables from the process environment; if you want local `.env` loading, run it through your shell or deployment tool.

The server handles `SIGINT` and `SIGTERM` with graceful shutdown. It stops accepting new requests and gives in-flight requests up to 10 seconds to complete.

## Run Tests

```bash
go test ./...
```

The tests include the challenge examples and the custom edge case:

```text
Pack sizes: 23, 31, 53
Amount: 500,000
Expected output: 23 x 2, 31 x 7, 53 x 9429
```

## Docker

Build and run:

```bash
docker build -t pack-calculator .
docker run --rm -p 8080:8080 pack-calculator
```

To persist pack-size changes outside the container:

```bash
docker run --rm -p 8080:8080 \
  -v "$(pwd)/data:/app/data" \
  pack-calculator
```

## API

### Get Pack Sizes

```http
GET /api/pack-sizes
```

Response:

```json
{
  "packSizes": [250, 500, 1000, 2000, 5000]
}
```

### Update Pack Sizes

```http
PUT /api/pack-sizes
Content-Type: application/json

{
  "packSizes": [23, 31, 53]
}
```

Pack sizes must be positive integers. Duplicates are removed and sizes are stored sorted.

### Calculate Packs

```http
POST /api/calculate
Content-Type: application/json

{
  "items": 500000
}
```

Response:

```json
{
  "itemsOrdered": 500000,
  "itemsShipped": 500000,
  "totalPacks": 9438,
  "packs": [
    { "packSize": 23, "quantity": 2 },
    { "packSize": 31, "quantity": 7 },
    { "packSize": 53, "quantity": 9429 }
  ]
}
```

## Algorithm

The calculator uses dynamic programming over reachable shipped totals up to `items ordered + largest pack size - 1`.

That range is enough because repeatedly using the largest pack always provides an achievable shipped total within that bound. The first reachable total at or above the order is therefore the minimum total number of items that can be shipped. The dynamic-programming state stores the fewest packs for each total, so the chosen total also satisfies the fewest-packs tie breaker.
