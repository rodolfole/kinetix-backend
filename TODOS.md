TODOS: 
Ver si es mejor tener ya definiddas las categorias de edad para que sean de 5 en 5 y de 10 en 10



# Desde kinetix-api/
go run ./cmd/seed/

# Migrations
goose -dir internal/adapters/postgresql/migrations postgres "postgres://postgres@localhost:5432/kinetix-db?sslmode=disable" up
goose -dir internal/adapters/postgresql/migrations postgres "postgres://postgres@localhost:5432/kinetix-db?sslmode=disable" down


# Utils
    Detener puerto
    netstat -ano | findstr :8080
    taskkill /PID 22820 /F

