package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kinetix-api/internal/admin"
	"kinetix-api/internal/agecategories"
	"kinetix-api/internal/auth"
	"kinetix-api/internal/config"
	"kinetix-api/internal/database"
	"kinetix-api/internal/distances"
	"kinetix-api/internal/events"
	"kinetix-api/internal/features/tags"
	kinetix_middleware "kinetix-api/internal/middleware"
	"kinetix-api/internal/mp"
	"kinetix-api/internal/orders"
	"kinetix-api/internal/organizers"
	"kinetix-api/internal/participants"
	"kinetix-api/internal/registrations"
	"kinetix-api/internal/routes"
	"kinetix-api/internal/runnerkits"
	"kinetix-api/internal/store"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// @title Kinetix API
// @version 1.0
// @description API for running sports events management
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Structured JSON logging (PLAN-16 T14): leveled, machine-readable.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Debug("logging configured")

	// Load configuration
	cfg := config.Load()

	// Initialize validator
	validator := validator.New()

	// Connect to database using pgx/v5
	pool, err := database.NewConnectionPool(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		DBName:   cfg.Database.DBName,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		slog.Error("database connection failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer database.ClosePool(pool)

	// Initialize Store (sqlc-based repository)
	appStore := store.NewStore(pool)

	// Initialize JWT config
	tokenExp, err := time.ParseDuration(cfg.JWT.TokenExpiry)
	if err != nil || tokenExp <= 0 {
		tokenExp = 24 * time.Hour
	}
	jwtCfg := auth.Config{
		SecretKey:     cfg.JWT.SecretKey,
		TokenExpiry:   tokenExp,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	// Initialize repositories (using sqlc Store)
	organizerRepo := organizers.NewRepository(appStore)
	eventRepo := events.NewRepository(appStore)
	participantRepo := participants.NewRepository(appStore)
	registrationRepo := registrations.NewRepository(appStore)
	orderRepo := orders.NewRepository(appStore)
	adminRepo := admin.NewRepository(appStore)

	// Initialize services
	organizerService := organizers.NewService(organizerRepo, validator)
	eventService := events.NewService(eventRepo, validator)
	participantService := participants.NewService(participantRepo, validator)
	registrationService := registrations.NewService(registrationRepo, registrationRepo, validator) // its ExecTx adapts store.ExecTx
	orderService := orders.NewService(orderRepo, adminRepo, orders.NewPaidTxRunner(appStore), validator)
	adminService := admin.NewService(adminRepo)
	mpClient := mp.NewClient(cfg.MercadoPago)

	// Initialize auth service and handler
	// JWT secret comes from config.Load() which panics if JWT_SECRET is not set
	authCfg := jwtCfg // single JWT configuration (was duplicated)
	authService := auth.NewService(appStore, authCfg)
	authHandler := auth.NewHandler(authService, validator)

	// Initialize distances feature
	distanceRepo := distances.NewRepository(appStore)
	distanceService := distances.NewService(distanceRepo, validator)
	distanceHandler := distances.NewHandler(distanceService, validator)

	// Initialize age categories feature
	ageCategoryRepo := agecategories.NewRepository(appStore)
	ageCategoryHandler := agecategories.NewHandler(ageCategoryRepo)

	// Initialize tags feature
	tagRepo := tags.NewRepository(appStore)
	tagHandler := tags.NewHandler(tagRepo)

	// Initialize runner kits feature
	runnerKitRepo := runnerkits.NewRepository(appStore)
	runnerKitService := runnerkits.NewService(runnerKitRepo, validator)
	runnerKitHandler := runnerkits.NewHandler(runnerKitService, validator)

	// Initialize routes feature
	routesRepo := routes.NewRepository(appStore)
	routesService := routes.NewService(routesRepo, validator)
	routesHandler := routes.NewHandler(routesService, validator)

	// Initialize handlers
	organizerHandler := organizers.NewHandler(organizerService, validator)
	eventHandler := events.NewHandler(eventService, validator)
	participantHandler := participants.NewHandler(participantService)
	registrationHandler := registrations.NewHandler(registrationService)
	orderHandler := orders.NewHandler(orderService, cfg.MercadoPago.WebhookSecret, mpClient) // payment webhooks confirm with MP

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	kinetix_middleware.RequestIDResolver = func(r *http.Request) string {
		if v := r.Context().Value(chimiddleware.RequestIDKey); v != nil {
			if id, ok := v.(string); ok {
				return id
			}
		}
		return ""
	}
	r.Use(kinetix_middleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(kinetix_middleware.CORS(cfg.Server.CORSAllowedOrigins))

	// Public routes
	r.Get("/health", healthCheck)
	r.Get("/", welcome)

	// Auth routes (public) with rate limiting
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(kinetix_middleware.AuthRateLimit())
		authHandler.Routes(r)
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Get("/events", eventHandler.ListEvents)
		r.Get("/events/{id}", eventHandler.GetEvent)
		r.Get("/events/slug/{slug}", eventHandler.GetEventBySlug)
		r.Get("/events/{id}/detail", eventHandler.GetEventDetail)

		// Distance types catalog (public)
		r.Get("/distances/types", distanceHandler.ListTypes)

		// Organizer public detail (for event listings)
		r.Get("/organizers/{id}", organizerHandler.GetOrganizer)

		// Age categories catalog (public)
		r.Get("/age-categories", ageCategoryHandler.List)

		// Event tags (public)
		r.Get("/events/{eventId}/tags", tagHandler.ListByEvent)

		// Event sub-resources (public)
		r.Get("/events/event/{event_id}/distances", distanceHandler.ListByEvent)
		r.Get("/events/distance/{distance_id}/pricing", eventHandler.ListPricingStages)
		r.Get("/events/distance/{distance_id}/pricing/current", eventHandler.GetCurrentPricing)
		r.Get("/events/event/{event_id}/sponsors", eventHandler.ListSponsors)
		r.Get("/events/event/{event_id}/locations", eventHandler.ListRegistrationLocations)
		r.Get("/events/event/{event_id}/runner-kits", runnerKitHandler.ListByEvent)
		r.Get("/events/event/{event_id}/routes", routesHandler.ListByEvent)
		r.Get("/events/routes/distance/{distance_id}", routesHandler.ListByDistance)
		r.Get("/events/{id}/field-config", eventHandler.GetFieldConfig)

		// Protected routes — individual routes only, NO subrouters at `/events`
		// to avoid Chi intercepting the public GET /events with JWT middleware.
		r.Group(func(r chi.Router) {
			r.Use(kinetix_middleware.JWTAuth(jwtCfg))

			// Organizers management (protected) — individual paths, no subrouter at /organizers
			// (GET /organizers/{id} is public at line 137)
			r.Post("/organizers", organizerHandler.CreateOrganizer)
			r.Get("/organizers", organizerHandler.ListOrganizers)
			r.Put("/organizers/{id}", organizerHandler.UpdateOrganizer)
			r.Delete("/organizers/{id}", organizerHandler.DeleteOrganizer)

			// Event management: organizers and admins only (participants get 403).
			r.Group(func(r chi.Router) {
				r.Use(kinetix_middleware.RequireAnyRole("organizer", "admin"))

				// Distances management (protected)
				r.Route("/distances", func(r chi.Router) {
					distanceHandler.Routes(r)
				})

				// Event management (protected) — individual paths, no subrouter at /events
				r.Post("/events", eventHandler.CreateEvent)
				r.Put("/events/{id}", eventHandler.UpdateEvent)
				r.Patch("/events/{id}/status", eventHandler.UpdateEventStatus)
				r.Delete("/events/{id}", eventHandler.DeleteEvent)

				r.Post("/events/awards", eventHandler.CreateAward)
				r.Get("/events/awards/{id}", eventHandler.GetAward)
				r.Put("/events/awards/{id}", eventHandler.UpdateAward)
				r.Delete("/events/awards/{id}", eventHandler.DeleteAward)

				r.Post("/events/pricing", eventHandler.CreatePricingStage)
				r.Get("/events/pricing/{id}", eventHandler.GetPricingStage)
				r.Put("/events/pricing/{id}", eventHandler.UpdatePricingStage)
				r.Delete("/events/pricing/{id}", eventHandler.DeletePricingStage)

				r.Post("/events/sponsors", eventHandler.CreateSponsor)
				r.Get("/events/sponsors/{id}", eventHandler.GetSponsor)
				r.Put("/events/sponsors/{id}", eventHandler.UpdateSponsor)
				r.Delete("/events/sponsors/{id}", eventHandler.DeleteSponsor)

				r.Post("/events/locations", eventHandler.CreateRegistrationLocation)
				r.Get("/events/locations/{id}", eventHandler.GetRegistrationLocation)
				r.Put("/events/locations/{id}", eventHandler.UpdateRegistrationLocation)
				r.Delete("/events/locations/{id}", eventHandler.DeleteRegistrationLocation)

				// Field config (protected) — togglable optional sections per event
				// GET is public (registered above); a second GET here would shadow it.
				r.Put("/events/{id}/field-config", eventHandler.UpdateFieldConfig)

				// Tags management (protected)
				r.Post("/events/tags", tagHandler.Create)
				r.Delete("/events/tags/{id}", tagHandler.Delete)

				// Runner kits management (protected)
				r.Route("/runner-kits", func(r chi.Router) {
					runnerKitHandler.Routes(r)
				})

				// Routes management (protected)
				r.Route("/routes", func(r chi.Router) {
					routesHandler.Routes(r)
				})
			})

			// Participants (protected)
			r.Route("/participants", func(r chi.Router) {
				participantHandler.Routes(r)
			})

			// Registrations (protected)
			r.Route("/registrations", func(r chi.Router) {
				registrationHandler.Routes(r)
			})

			// Orders (protected)
			r.Route("/orders", func(r chi.Router) {
				orderHandler.Routes(r)
			})

			// Admin (protected, requires role=admin)
			adminHandler := admin.NewHandlerWithMP(adminService, validator, mpClient, cfg.JWT.SecretKey) // sign OAuth states with the same secret
			r.Group(func(r chi.Router) {
				r.Use(kinetix_middleware.RequireRole("admin"))
				r.Get("/admin/config", adminHandler.GetConfig)
				r.Put("/admin/config", adminHandler.UpdateConfig)
				r.Get("/admin/payouts", adminHandler.ListPayouts)
				r.Get("/admin/organizers/{id}/connect-mp", adminHandler.ConnectMP)
				r.Post("/admin/organizers/callback", adminHandler.OAuthCallback)
				r.Post("/admin/payouts/{id}/send", adminHandler.SendPayout)
			})
		})

		// Webhook routes (no auth required)
		r.Post("/orders/webhook", orderHandler.PaymentWebhook)
	})

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting server", slog.String("addr", addr))

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("server exited cleanly")
}

// healthCheck handles GET /health
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

// welcome handles GET /
func welcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"message": "Welcome to Kinetix API",
		"version": "1.0.0",
		"docs": "/swagger/index.html"
	}`))
}
