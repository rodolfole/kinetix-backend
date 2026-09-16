# syntax=docker/dockerfile:1.6

# ---------- build stage ----------
FROM golang:1.25.5-alpine AS build
WORKDIR /src

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Install goose for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

# Copy source
COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/kinetix-api ./cmd/api

# Build seed CLI
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/kinetix-seed ./cmd/seed

# Build db-init helper (creates database before migrations)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/kinetix-dbinit ./scripts

# ---------- runtime stage ----------
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata curl postgresql-client && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/kinetix-api /app/kinetix-api
COPY --from=build /out/kinetix-seed /app/kinetix-seed
COPY --from=build /out/kinetix-dbinit /app/kinetix-dbinit
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY internal/adapters/postgresql/migrations /app/migrations

ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=8080 \
    DB_HOST=postgres \
    DB_PORT=5432 \
    DB_NAME=kinetix-db \
    DB_USER=postgres \
    DB_PASSWORD=postgres \
    DB_SSLMODE=disable \
    GIN_MODE=release

EXPOSE 8080

# default: run API; docker-compose overrides command for init containers
CMD ["/app/kinetix-api"]
