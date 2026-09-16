# Kinetix API

API for running sports events management built with Go, PostgreSQL, and Vertical Slice Architecture.

## Features

- **Organizers Management**: Create and manage event organizers
- **Events Management**: Full CRUD for events with distances, awards, pricing stages, sponsors, and registration locations
- **Participants Management**: Register participants with emergency contacts and medical info
- **Registrations**: Link participants to events and distances
- **Orders & Payments**: MercadoPago integration (mock) for payment processing
- **JWT Authentication**: Secure API endpoints
- **Vertical Slice Architecture**: Clean, maintainable code organization

## Tech Stack

- **Language**: Go 1.21+
- **Router**: chi
- **Database**: PostgreSQL
- **Query Builder**: sqlc
- **Migrations**: goose
- **Authentication**: golang-jwt/jwt
- **Validation**: go-playground/validator
- **Documentation**: swaggo/swag

## Project Structure

```
kinetix-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── auth/                    # JWT authentication
│   ├── config/                  # Configuration management
│   ├── database/
│   │   ├── migrations/          # Goose migrations
│   │   ├── queries/             # SQLC queries
│   │   └── store/               # Store package
│   ├── events/                  # Events vertical slice
│   ├── json/                    # JSON helpers
│   ├── middleware/              # HTTP middleware
│   ├── orders/                  # Orders vertical slice
│   ├── participants/            # Participants vertical slice
│   ├── registrations/           # Registrations vertical slice
│   └── organizers/              # Organizers vertical slice
├── sqlc.yaml                    # SQLC configuration
└── go.mod
```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- goose (for migrations)
- sqlc (for code generation)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd kinetix-api
```

2. Install dependencies:
```bash
go mod tidy
```

3. Install tools:
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

4. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

5. Run migrations:
```bash
cd internal/database/migrations
goose postgres "user=postgres password= dbname=kinetix-db host=localhost port=5432 sslmode=disable" up
```

6. Generate SQLC code:
```bash
sqlc generate
```

7. Run the server:
```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:8080`

## Environment Variables

```env
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=kinetix-db
DB_USER=postgres
DB_PASSWORD=
DB_SSLMODE=disable

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_TOKEN_EXPIRY=24h

# MercadoPago (optional)
MP_ACCESS_TOKEN=
MP_PUBLIC_KEY=
```

## API Endpoints

### Public Endpoints

- `GET /health` - Health check
- `GET /api/v1/events` - List events
- `GET /api/v1/events/{id}` - Get event details
- `GET /api/v1/events/slug/{slug}` - Get event by slug
- `GET /api/v1/events/{id}/detail` - Get full event detail with distances, awards, pricing, sponsors, locations
- `POST /api/v1/orders/webhook` - MercadoPago webhook

### Protected Endpoints (requires JWT)

#### Organizers
- `POST /api/v1/organizers` - Create organizer
- `GET /api/v1/organizers/{id}` - Get organizer
- `GET /api/v1/organizers` - List organizers
- `PUT /api/v1/organizers/{id}` - Update organizer
- `DELETE /api/v1/organizers/{id}` - Delete organizer

#### Events Management
- `POST /api/v1/events` - Create event
- `PUT /api/v1/events/{id}` - Update event
- `PATCH /api/v1/events/{id}/status` - Update event status
- `DELETE /api/v1/events/{id}` - Delete event

#### Distances
- `POST /api/v1/events/distances` - Create distance
- `PUT /api/v1/events/distances/{id}` - Update distance
- `DELETE /api/v1/events/distances/{id}` - Delete distance

#### Awards
- `POST /api/v1/events/awards` - Create award
- `PUT /api/v1/events/awards/{id}` - Update award
- `DELETE /api/v1/events/awards/{id}` - Delete award

#### Pricing Stages
- `POST /api/v1/events/pricing` - Create pricing stage
- `PUT /api/v1/events/pricing/{id}` - Update pricing stage
- `DELETE /api/v1/events/pricing/{id}` - Delete pricing stage

#### Sponsors
- `POST /api/v1/events/sponsors` - Create sponsor
- `PUT /api/v1/events/sponsors/{id}` - Update sponsor
- `DELETE /api/v1/events/sponsors/{id}` - Delete sponsor

#### Registration Locations
- `POST /api/v1/events/locations` - Create registration location
- `PUT /api/v1/events/locations/{id}` - Update registration location
- `DELETE /api/v1/events/locations/{id}` - Delete registration location

#### Participants
- `POST /api/v1/participants` - Create participant
- `GET /api/v1/participants/{id}` - Get participant
- `GET /api/v1/participants` - List participants
- `PUT /api/v1/participants/{id}` - Update participant
- `DELETE /api/v1/participants/{id}` - Delete participant
- `POST /api/v1/participants/{id}/sign-waiver` - Sign waiver

#### Registrations
- `POST /api/v1/registrations` - Create registration
- `GET /api/v1/registrations/{id}` - Get registration
- `GET /api/v1/registrations/event/{event_id}` - List registrations by event
- `GET /api/v1/registrations/participant/{participant_id}` - List registrations by participant
- `POST /api/v1/registrations/{id}/confirm` - Confirm registration
- `POST /api/v1/registrations/{id}/cancel` - Cancel registration
- `DELETE /api/v1/registrations/{id}` - Delete registration

#### Orders
- `POST /api/v1/orders` - Create order with payment
- `GET /api/v1/orders/{id}` - Get order
- `GET /api/v1/orders/participant/{participant_id}` - List orders by participant
- `GET /api/v1/orders/event/{event_id}` - List orders by event
- `POST /api/v1/orders/{id}/cancel` - Cancel order

## Database Schema

### Organizers
- id, business_name, brand_name, rfc, billing_address, contact info

### Events
- id, organizer_id, name, slug, description, date, deadline, location, status

### Distances
- id, event_id, km, capacity, surface, GPS coordinates

### Awards
- id, distance_id, age_range, gender, prize_amount

### Pricing Stages
- id, distance_id, name, price, start_date, end_date

### Participants
- id, personal info, emergency contact, medical info, waiver_signed_at

### Registrations
- id, event_id, participant_id, distance_id, category, time, status

### Orders
- id, registration_id, participant_id, event_id, amount, status, payment info

### Sponsors
- id, event_id, name, tier, logo_url, website_url

### Registration Locations
- id, event_id, location_name, address, schedule

## Development

### Run Migrations

```bash
cd internal/database/migrations
goose postgres "connection-string" up
```

### Generate SQLC Code

```bash
sqlc generate
```

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o bin/kinetix-api ./cmd/api
```

## License

MIT
