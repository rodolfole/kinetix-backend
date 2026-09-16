# Quick Start Guide - Kinetix API

## Prerequisites Installed
✅ Go module initialized
✅ All dependencies installed
✅ Project structure created
✅ Database migrations created
✅ SQLC queries configured
✅ Complete API implementation
✅ Build successful

## Next Steps

### 1. Install Required Tools

```bash
# Install goose for migrations
go install github.com/pressly/goose/v3/cmd/goose@latest

# Install sqlc for code generation
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### 2. Set Up PostgreSQL Database

Create the database:
```bash
# Connect to PostgreSQL
psql -U postgres -h localhost

# Create database
CREATE DATABASE "kinetix-db";
\q
```

### 3. Run Database Migrations

```bash
cd internal/database/migrations
goose postgres "user=postgres password= dbname=kinetix-db host=localhost port=5432 sslmode=disable" up
```

To rollback a migration:
```bash
goose postgres "user=postgres password= dbname=kinetix-db host=localhost port=5432 sslmode=disable" down
```

### 4. Generate SQLC Code (Optional)

If you want to use sqlc-generated code instead of manual queries:
```bash
cd C:\Users\codeqix\Downloads\kinetix-api
sqlc generate
```

This will generate Go code in `internal/database/store/` based on the SQL queries.

### 5. Configure Environment Variables

```bash
# Copy the example file
copy .env.example .env

# Edit .env with your settings (optional, defaults are provided)
```

### 6. Run the API Server

```bash
go run cmd/api/main.go
```

The server will start at `http://localhost:8080`

### 7. Test the API

Health check:
```bash
curl http://localhost:8080/health
```

Welcome endpoint:
```bash
curl http://localhost:8080/
```

List events (public):
```bash
curl http://localhost:8080/api/v1/events
```

### 8. Generate JWT Token for Protected Routes

You'll need to create a JWT token to access protected endpoints. Here's a Go snippet to generate one:

```go
package main

import (
    "fmt"
    "time"
    "kinetix-api/internal/auth"
)

func main() {
    cfg := auth.Config{
        SecretKey:   "your-secret-key-change-in-production",
        TokenExpiry: 24 * time.Hour,
    }
    
    token, err := auth.GenerateToken(cfg, "user-123", "admin")
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Token:", token)
}
```

### 9. Test Protected Endpoints

```bash
# Create an organizer
curl -X POST http://localhost:8080/api/v1/organizers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "business_name": "Eventos Deportivos del Norte S.A.",
    "brand_name": "Chihuahua Race Team",
    "rfc": "EDN220101XXX",
    "billing_zip_code": "33000",
    "billing_state": "Chihuahua",
    "billing_city": "Delicias",
    "contact_email": "soporte@chihuahuarace.com",
    "contact_phone": "6391234567"
  }'
```

## Project Structure Summary

```
kinetix-api/
├── cmd/api/main.go                 # Application entry point
├── internal/
│   ├── auth/                       # JWT authentication
│   ├── config/                     # Environment configuration
│   ├── database/
│   │   ├── migrations/             # 11 migration files
│   │   ├── queries/                # 10 SQL query files
│   │   └── store/                  # Store package with transactions
│   ├── json/                       # JSON helper utilities
│   ├── middleware/                 # HTTP middleware (JWT, CORS)
│   ├── organizers/                 # Organizers vertical slice
│   ├── events/                     # Events vertical slice
│   ├── participants/               # Participants vertical slice
│   ├── registrations/              # Registrations vertical slice
│   └── orders/                     # Orders with MercadoPago mock
├── sqlc.yaml                       # SQLC configuration
├── go.mod                          # Go dependencies
└── README.md                       # Full documentation
```

## Database Tables Created

1. **organizers** - Event organizers with billing and contact info
2. **events** - Running events with status tracking
3. **distances** - Race distances (5K, 10K, etc.) with capacity and GPS
4. **awards** - Prize awards by age range and gender
5. **pricing_stages** - Dynamic pricing by date ranges
6. **participants** - Runner profiles with emergency contacts
7. **registrations** - Event registrations linking participants to distances
8. **orders** - Payment orders with MercadoPago integration
9. **sponsors** - Event sponsors by tier
10. **registration_locations** - Physical registration locations

## Key Features Implemented

✅ Vertical Slice Architecture - Each domain has handler/service/repository
✅ PostgreSQL with Goose migrations
✅ SQLC query configuration
✅ Store pattern with transaction support
✅ JWT authentication middleware
✅ Request validation with go-playground/validator
✅ JSON helper utilities for consistent responses
✅ MercadoPago mock payment integration
✅ CORS support
✅ Graceful shutdown
✅ Health check endpoint
✅ Complete CRUD operations for all entities

## API Endpoints

### Public (No Auth Required)
- `GET /health` - Health check
- `GET /api/v1/events` - List events
- `GET /api/v1/events/{id}` - Get event
- `GET /api/v1/events/slug/{slug}` - Get event by slug
- `GET /api/v1/events/{id}/detail` - Full event detail
- `POST /api/v1/orders/webhook` - Payment webhook

### Protected (JWT Required)
- **Organizers**: Full CRUD
- **Events**: Full CRUD + sub-resources (distances, awards, pricing, sponsors, locations)
- **Participants**: Full CRUD + waiver signing
- **Registrations**: Create, list, confirm, cancel
- **Orders**: Create with payment, list, cancel

## Additional Recommendations

### Database Enhancements to Consider:
1. **Indexes**: Already added for performance on frequently queried fields
2. **Triggers**: Auto-update `updated_at` timestamps
3. **Constraints**: Foreign keys, check constraints on enums
4. **Unique Constraints**: Email, RFC, event slug

### Future Enhancements:
1. **QR Code Generation**: For participant check-in
2. **Email Notifications**: Registration confirmations
3. **Real MercadoPago Integration**: Replace mock with actual SDK
4. **File Upload**: Event logos, participant photos
5. **Analytics Dashboard**: Registration stats, revenue reports
6. **Waitlist Management**: When capacity is reached
7. **Discount Codes**: Promotional pricing
8. **Team Registrations**: Bulk registration for groups
9. **Bib Number Assignment**: Automatic race bib generation
10. **Results Publishing**: Post-event race results
