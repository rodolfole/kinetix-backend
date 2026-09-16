package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"kinetix-api/internal/config"
	"kinetix-api/internal/database"
	"kinetix-api/internal/database/convert"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/distances"
	"kinetix-api/internal/events"
	"kinetix-api/internal/organizers"
	"kinetix-api/internal/participants"
	"kinetix-api/internal/registrations"
	"kinetix-api/internal/store"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Seeder populates the database with test data
type Seeder struct {
	pool      *pgxpool.Pool
	store     *store.Store
	validator *validator.Validate
}

func main() {
	log.Println("Starting database seeder...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	pool, err := database.NewConnectionPool(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		DBName:   cfg.Database.DBName,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.ClosePool(pool)

	// Initialize store
	appStore := store.NewStore(pool)
	validator := validator.New()

	seeder := &Seeder{
		pool:      pool,
		store:     appStore,
		validator: validator,
	}

	// Run seeding
	if err := seeder.Run(); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Seeding completed successfully!")
}

func (s *Seeder) Run() error {
	ctx := context.Background()

	// Create organizer (idempotent — skip if RFC already exists)
	log.Println("Creating organizer...")
	organizer, err := s.createOrganizer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create organizer: %w", err)
	}
	log.Printf("  Organizer: %s (%s)", organizer.BusinessName, organizer.ID)

	// Create event (idempotent — skip if slug already exists)
	log.Println("Creating event...")
	event, err := s.createEvent(ctx, organizer.ID)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	log.Printf("  Event: %s (%s)", event.Name, event.ID)

	// Create runner kit items for the main event
	s.createRunnerKits(ctx, event.ID)

	// Distance types are seeded by migration (003_create_distances.sql)
	// Also fallback: seed them inline in case migration is out of date
	log.Println("Loading distance types...")
	distanceTypes, err := s.store.Queries.ListDistanceTypes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list distance types: %w", err)
	}
	if len(distanceTypes) == 0 {
		log.Println("  No distance types found, seeding inline...")
		typeDef := []struct {
			name string
			km   float64
		}{
			{"5K", 5},
			{"10K", 10},
			{"15K", 15},
			{"21K Half Marathon", 21.1},
			{"42K Marathon", 42.2},
		}
		for _, d := range typeDef {
			dt, err := s.store.Queries.CreateDistanceType(ctx, db.CreateDistanceTypeParams{
				Name: d.name,
				Km:   convert.ToPgNumeric(d.km),
			})
			if err != nil {
				log.Printf("  Warning: failed to seed distance type %s: %v", d.name, err)
				continue
			}
			distanceTypes = append(distanceTypes, dt)
			log.Printf("  Seeded distance type inline: %s (%.1fkm)", dt.Name, convert.PgNumericToFloat64(dt.Km))
		}
		if len(distanceTypes) == 0 {
			return fmt.Errorf("failed to seed any distance types")
		}
	}
	for _, dt := range distanceTypes {
		log.Printf("  Loaded distance type: %s (%.1fkm)", dt.Name, convert.PgNumericToFloat64(dt.Km))
	}

	// Create distances using distance types
	log.Println("Creating distances...")
	createdDistances, err := s.createDistances(ctx, event.ID, distanceTypes)
	if err != nil {
		return fmt.Errorf("failed to create distances: %w", err)
	}
	if len(createdDistances) == 0 {
		return fmt.Errorf("no distances were created — check that distance_types are seeded in the database (run migrations)")
	}

	// Create pricing stages
	log.Println("Creating pricing stages...")
	if err := s.createPricingStages(ctx, createdDistances); err != nil {
		return fmt.Errorf("failed to create pricing stages: %w", err)
	}

	// Create participants
	log.Println("Creating participants...")
	participantRepo := participants.NewRepository(s.store)
	partService := participants.NewService(participantRepo, s.validator)

	// Sample participant data
	participantData := []struct {
		firstName string
		lastName  string
		email     string
	}{
		{"Juan", "Pérez", "juan.perez@example.com"},
		{"María", "García", "maria.garcia@example.com"},
		{"Carlos", "Rodríguez", "carlos.rodriguez@example.com"},
		{"Ana", "López", "ana.lopez@example.com"},
		{"Pedro", "Martínez", "pedro.martinez@example.com"},
	}

	var createdParticipants []participants.Participant
	for _, p := range participantData {
		// Check if participant already exists by email
		existing, err := partService.GetByEmail(ctx, p.email)
		if err == nil && existing.ID != uuid.Nil {
			createdParticipants = append(createdParticipants, existing)
			log.Printf("  Skipping participant — already exists: %s %s (%s)", existing.FirstName, existing.LastName, existing.ID)
			continue
		}

		now := time.Now()
		birthDate := now.AddDate(-25, 0, 0) // 25 years ago

		secondLastName := "Smith"
		municipality := "Guadalajara"
		state := "Jalisco"
		zipCode := "44100"
		team := "Runners Club"
		size := "M"

		req := participants.CreateParticipantRequest{
			FirstName:      p.firstName,
			LastName:       p.lastName,
			SecondLastName: &secondLastName,
			Gender:         "male",
			BirthDate:      birthDate,
			Email:          p.email,
			Phone:          "+52 555 123 4567",
			Country:        "MX",
			Municipality:   &municipality,
			State:          &state,
			ZipCode:        &zipCode,
			Team:           &team,
			Size:           &size,
		}

		participant, err := partService.Create(ctx, req)
		if err != nil {
			// Last resort: try to look up by email in case check above missed it
			existing, lookupErr := partService.GetByEmail(ctx, p.email)
			if lookupErr == nil && existing.ID != uuid.Nil {
				createdParticipants = append(createdParticipants, existing)
				log.Printf("  Recovered existing participant: %s %s (%s)", existing.FirstName, existing.LastName, existing.ID)
				continue
			}
			log.Printf("  Warning: failed to create participant %s: %v", p.email, err)
			continue
		}
		createdParticipants = append(createdParticipants, participant)
		log.Printf("  Created participant: %s %s (%s)", participant.FirstName, participant.LastName, participant.ID)
	}

	if len(createdParticipants) == 0 {
		log.Println("  No participants created, skipping registrations")
		return nil
	}

	// Create registrations with bib numbers
	log.Println("Creating registrations...")
	regRepo := registrations.NewRepository(s.store)
	regService := registrations.NewService(regRepo, s.store, s.validator)

	// Check which participants already have a registration for this event
	type participantReg struct {
		participants.Participant
		distance db.Distance
	}
	var toRegister []participantReg
	for i, p := range createdParticipants {
		existingRegs, err := regService.ListByParticipant(ctx, p.ID)
		if err == nil {
			alreadyRegistered := false
			for _, reg := range existingRegs {
				if reg.EventID == event.ID {
					log.Printf("  Skipping registration — %s %s already registered for this event (bib: %s)",
						p.FirstName, p.LastName, *reg.BibNumber)
					alreadyRegistered = true
					break
				}
			}
			if alreadyRegistered {
				continue
			}
		}
		distanceIdx := i % len(createdDistances)
		toRegister = append(toRegister, participantReg{
			Participant: p,
			distance:    createdDistances[distanceIdx],
		})
	}

	for _, pr := range toRegister {
		req := registrations.CreateRegistrationRequest{
			EventID:       event.ID,
			ParticipantID: pr.Participant.ID,
			DistanceID:    pr.distance.ID,
			Category:      " libre",
		}

		reg, err := regService.Create(ctx, req)
		if err != nil {
			log.Printf("  Warning: failed to create registration for %s: %v", pr.Email, err)
			continue
		}

		log.Printf("  Created registration: %s %s -> Bib: %s, QR: %s",
			pr.FirstName, pr.LastName, *reg.BibNumber, *reg.QRCode)
	}

	// Create auth users (with hashed passwords) — skip if email already exists
	log.Println("Creating auth users...")

	defaultPassword := "seed1234"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashBytes)

	type userDef struct {
		email        string
		role         string
		organizerID  *uuid.UUID
		participantID *uuid.UUID
	}

	var userDefs []userDef
	userDefs = append(userDefs, userDef{
		email:       "contact@kinetix.com",
		role:        "organizer",
		organizerID: &organizer.ID,
	})
	for _, p := range createdParticipants {
		p := p
		userDefs = append(userDefs, userDef{
			email:          p.Email,
			role:           "participant",
			participantID: &p.ID,
		})
	}

	for _, u := range userDefs {
		_, lookupErr := s.store.Queries.GetUserByEmail(ctx, u.email)
		if lookupErr == nil {
			log.Printf("  Skipping user — already exists: %s", u.email)
			continue
		}

		params := db.CreateUserParams{
			Email:        u.email,
			PasswordHash: hashedPassword,
			Role:         u.role,
		}
		if u.organizerID != nil {
			params.OrganizerID = convert.ToPgUUID(*u.organizerID)
		}
		if u.participantID != nil {
			params.ParticipantID = convert.ToPgUUID(*u.participantID)
		}

		if _, createErr := s.store.Queries.CreateUser(ctx, params); createErr != nil {
			log.Printf("  Warning: failed to create user %s: %v", u.email, createErr)
			continue
		}
		log.Printf("  Created user: %s / %s", u.email, defaultPassword)
	}

	// Create additional mock events (10K Valencia Marathon & THE NEON DASH)
	log.Println("Creating additional mock events from legacy mock data...")
	if err := s.createAdditionalMockEvents(ctx, distanceTypes); err != nil {
		return fmt.Errorf("failed to create additional mock events: %w", err)
	}

	log.Println("Seeding completed!")
	return nil
}

func (s *Seeder) createOrganizer(ctx context.Context) (organizers.Organizer, error) {
	repo := organizers.NewRepository(s.store)
	service := organizers.NewService(repo, s.validator)

	// Check if already exists by RFC
	existing, err := repo.GetByRFC(ctx, "KIN210101ABC")
	if err == nil && existing.ID != uuid.Nil {
		log.Printf("  Skipping organizer — already exists (RFC: %s)", existing.Rfc)
		return existing, nil
	}

	req := organizers.CreateOrganizerRequest{
		BusinessName:   "Kinetix Events S.A. de C.V.",
		BrandName:      "Kinetix",
		RFC:            "KIN210101ABC",
		BillingZipCode: "44100",
		BillingState:   "Jalisco",
		BillingCity:    "Guadalajara",
		ContactName:    "Juan Perez",
		ContactEmail:   "contact@kinetix.com",
		ContactPhone:   "+52 33 3333 3333",
		LogoUrl:        "https://example.com/logo.png",
	}

	org, err := service.Create(ctx, req)
	if err != nil {
		return organizers.Organizer{}, fmt.Errorf("failed to create organizer: %w", err)
	}
	log.Printf("  Created new organizer: %s (%s)", org.BusinessName, org.ID)
	return org, nil
}

func (s *Seeder) createEvent(ctx context.Context, organizerID uuid.UUID) (events.Event, error) {
	repo := events.NewRepository(s.store)
	service := events.NewService(repo, s.validator)

	// Check if already exists by slug
	existing, err := repo.GetBySlug(ctx, "carrera-kinetix-2026")
	if err == nil && existing.ID != uuid.Nil {
		log.Printf("  Skipping event — already exists (slug: %s)", existing.Slug)
		return existing, nil
	}

	eventDate := time.Now().AddDate(0, 2, 0) // 2 months from now
	deadline := eventDate.AddDate(0, -1, 0)  // 1 month before event

	req := events.CreateEventRequest{
		OrganizerID:  organizerID,
		Name:         "Carrera Kinetix 2026",
		Slug:         "carrera-kinetix-2026",
		Description:  "La carrera más grande de Jalisco. Únete a más de 1000 corredores.",
		EventDate:    eventDate,
		Deadline:     deadline,
		State:        "Jalisco",
		Municipality: "Guadalajara",
		Address:      "Av. Morelos 100, Centro, Guadalajara, Jalisco",
		Judges:       "Juez Principal: Dr. Roberto Sánchez",
		Rules:        "Categoría libre: abierta a todos los participantes",
		Risks:        "Riesgos mínimos, hydrated areas available",
		Transit:      "Estacionamiento disponible en zona",
		Itinerary: []events.ItineraryItem{
			{Key: "Salida", Time: "07:00", Icon: "flag"},
			{Key: "Meta", Time: "10:00", Icon: "finish"},
		},
	}

	evt, err := service.Create(ctx, req)
	if err != nil {
		return events.Event{}, fmt.Errorf("failed to create event: %w", err)
	}
	log.Printf("  Created new event: %s (%s)", evt.Name, evt.ID)
	return evt, nil
}

func (s *Seeder) createDistances(ctx context.Context, eventID uuid.UUID, distanceTypes []db.DistanceType) ([]db.Distance, error) {
	distRepo := distances.NewRepository(s.store)
	distService := distances.NewService(distRepo, s.validator)

	// Check if distances already exist for this event
	existing, err := distService.ListByEvent(ctx, eventID)
	if err == nil && len(existing) > 0 {
		log.Printf("  Skipping distances — %d already exist for this event", len(existing))
		return existing, nil
	}

	// Map km -> distance type (use string key to avoid float64 precision issues)
	typeByKm := make(map[string]db.DistanceType)
	for _, dt := range distanceTypes {
		km := fmt.Sprintf("%.2f", convert.PgNumericToFloat64(dt.Km))
		typeByKm[km] = dt
	}

	// Define event-specific distances with capacity/surface
	typeDefs := []struct {
		km       float64
		capacity int
		surface  string
	}{
		{5, 300, "asphalt"},
		{10, 400, "asphalt"},
		{15, 200, "asphalt"},
		{21.1, 200, "asphalt"},
		{42.2, 100, "asphalt"},
	}

	var created []db.Distance
	for _, d := range typeDefs {
		dt, ok := typeByKm[fmt.Sprintf("%.2f", d.km)]
		if !ok {
			log.Printf("  Warning: distance type for %.1fkm not found, skipping", d.km)
			continue
		}

		req := distances.CreateDistanceRequest{
			EventID:        eventID,
			DistanceTypeID: dt.ID,
			Capacity:       d.capacity,
			Surface:        d.surface,
		}

		distance, err := distService.Create(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create distance %.1fkm: %w", d.km, err)
		}
		created = append(created, distance)
	}

	return created, nil
}

func (s *Seeder) createAdditionalMockEvents(ctx context.Context, distanceTypes []db.DistanceType) error {
	// Build km -> distance type map
	typeByKm := make(map[string]db.DistanceType)
	for _, dt := range distanceTypes {
		km := fmt.Sprintf("%.2f", convert.PgNumericToFloat64(dt.Km))
		typeByKm[km] = dt
	}

	// Helper: find distance type by km (fuzzy match)
	findType := func(km float64) (db.DistanceType, bool) {
		dt, ok := typeByKm[fmt.Sprintf("%.2f", km)]
		return dt, ok
	}

	// ─── EVENT 1: 10K Valencia Marathon ────────────────────────────────
	log.Println("  Setting up '10K Valencia Marathon'...")

	org1, err := s.createOrGetOrganizer(ctx, "VRA240101ABC", "Valencia Running Association",
		"Valencia Running Association", "contact@valenciarunning.com", "+34 96 123 4567",
		"Valencia", "Valencia", "46001")
	if err != nil {
		return fmt.Errorf("failed to setup Valencia organizer: %w", err)
	}

	valenciaDate := time.Date(2024, 12, 1, 8, 0, 0, 0, time.UTC)
	valenciaDeadline := time.Date(2024, 11, 15, 23, 59, 59, 0, time.UTC)

	evt1, err := s.createOrGetEvent(ctx, org1.ID, "10K Valencia Marathon", "10k-valencia-marathon",
		"Valencia is known worldwide as the City of Running. The 10K Valencia Marathon offers an exceptionally flat course designed for performance, winding through the historic center and finishing at the iconic City of Arts and Sciences.",
		valenciaDate, valenciaDeadline,
		"Valencia", "Valencia", "Ciudad de las Artes y las Ciencias, Valencia, Spain",
		"https://lh3.googleusercontent.com/aida-public/AB6AXuAkbDag8tf8bPkhAlsM1TKR17llHBVSnRFRIUzw_RYhu34WTOG11Sxjl1dXDL4o45KanfiyM0OkMLCA-Bjobsj5tfktHYh9kXGZQFjINTzukdhvC8cN2kjhUmkIoJp5jwYQ_KhtxTwiNBCrcyzfuxkrAObT36TLgGopof98bY_D0dpOUVmp9Ktlp9B-W0-4kyLF5cUh7gLAzy0DlDxelH8N0gk8BtaN5qC59OqapEf_BJWICnWzfc9bFmSPbgZlL9aAGQF2YUnf1PEQ",
		[]events.ItineraryItem{
			{Key: "Salida", Time: "08:00", Icon: "flag"},
			{Key: "Meta", Time: "11:00", Icon: "finish"},
		})
	if err != nil {
		return fmt.Errorf("failed to create Valencia event: %w", err)
	}
	s.createRunnerKits(ctx, evt1.ID)

	// Check if distances already exist for this event
	distRepo := distances.NewRepository(s.store)
	distService := distances.NewService(distRepo, s.validator)
	existingDists, _ := distService.ListByEvent(ctx, evt1.ID)

	var valenciaDistances []db.Distance
	if len(existingDists) > 0 {
		log.Printf("    Skipping distances — %d already exist", len(existingDists))
		valenciaDistances = existingDists
	} else {
		// Create distances: 5K Sprint, 10K Classic, Full Marathon
		typeDefs := []struct {
			km       float64
			capacity int
		}{
			{5, 5000},
			{10, 20000},
			{42.2, 5000},
		}
		for _, d := range typeDefs {
			dt, ok := findType(d.km)
			if !ok {
				log.Printf("    Warning: distance type for %.1fkm not found, skipping", d.km)
				continue
			}
			req := distances.CreateDistanceRequest{
				EventID:        evt1.ID,
				DistanceTypeID: dt.ID,
				Capacity:       d.capacity,
				Surface:        "asphalt",
			}
			dist, err := distService.Create(ctx, req)
			if err != nil {
				log.Printf("    Warning: failed to create distance %.1fkm: %v", d.km, err)
				continue
			}
			valenciaDistances = append(valenciaDistances, dist)
		}
		log.Printf("    Created %d distances for Valencia Marathon", len(valenciaDistances))
	}

	// Pricing stages for Valencia
	if len(valenciaDistances) > 0 {
		for _, d := range valenciaDistances {
			existingStages, _ := s.store.Queries.ListPricingStagesByDistance(ctx, d.ID)
			if len(existingStages) > 0 {
				log.Printf("    Skipping pricing stages — already exist for distance %s", d.ID)
				continue
			}
			basePrice := 4500.0
			// Early bird (10% discount)
			if _, err := s.store.Queries.CreatePricingStage(ctx, db.CreatePricingStageParams{
				DistanceID:  d.ID,
				Name:        pgtype.Text{String: "Early Bird", Valid: true},
				Price:       convert.ToPgNumeric(basePrice * 0.9),
				StartDate:   time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC),
				EndDate:     time.Date(2024, 10, 1, 23, 59, 59, 0, time.UTC),
				IsEarlyBird: pgtype.Bool{Bool: true, Valid: true},
			}); err != nil {
				log.Printf("    Warning: failed early bird pricing: %v", err)
			}
			// Regular
			if _, err := s.store.Queries.CreatePricingStage(ctx, db.CreatePricingStageParams{
				DistanceID:  d.ID,
				Name:        pgtype.Text{String: "Regular", Valid: true},
				Price:       convert.ToPgNumeric(basePrice),
				StartDate:   time.Date(2024, 10, 2, 0, 0, 0, 0, time.UTC),
				EndDate:     valenciaDate,
				IsEarlyBird: pgtype.Bool{Bool: false, Valid: true},
			}); err != nil {
				log.Printf("    Warning: failed regular pricing: %v", err)
			}
		}
	}

	// ─── EVENT 2: THE NEON DASH ────────────────────────────────────────
	log.Println("  Setting up 'THE NEON DASH'...")

	org2, err := s.createOrGetOrganizer(ctx, "NRC240101ABC", "Night Running Club",
		"Night Running Club", "contact@nightrunning.com", "+1 555 123 4567",
		"Metro", "Downtown", "10001")
	if err != nil {
		return fmt.Errorf("failed to setup Night Running organizer: %w", err)
	}

	neonDate := time.Date(2024, 10, 24, 20, 0, 0, 0, time.UTC)
	neonDeadline := time.Date(2024, 10, 17, 23, 59, 59, 0, time.UTC)

	evt2, err := s.createOrGetEvent(ctx, org2.ID, "THE NEON DASH", "neon-dash-10k",
		"Experience the pulse of the city under the neon lights. A technical night race designed for personal bests through the vibrant downtown metro area.",
		neonDate, neonDeadline,
		"Downtown Metro", "Metro", "123 Main Street, Downtown",
		"https://lh3.googleusercontent.com/aida-public/AB6AXuCYH6DOVlBJWUsvAF_JqMgQtVr0HHqMQ4_IlH-1FctEtOufriE934psPoSEIiV0Kyhw8pMkz8FrJ9hHaQvx_D6Y5Xj-_oDjFWftat07SB11uazBDUqm7fmesC_xPs_pGrEZikGjV-d3sN7_zfGNojqYRKvy3lFlF13u1JaEQdcltDFLg9KlA5OZ8uGu7YUVfhKn5RwUvDGo7MJQifEs57r67n3k33958_OOnixURtzIKQMzZ6uh4o1qc2oBx_FQbFgXqI1BhTZyDg42",
		[]events.ItineraryItem{
			{Key: "Salida", Time: "20:00", Icon: "flag"},
			{Key: "Meta", Time: "22:00", Icon: "finish"},
		})
	if err != nil {
		return fmt.Errorf("failed to create Neon Dash event: %w", err)
	}
	s.createRunnerKits(ctx, evt2.ID)

	// Check if distances already exist for this event
	existingDists2, _ := distService.ListByEvent(ctx, evt2.ID)
	if len(existingDists2) > 0 {
		log.Printf("    Skipping distances — %d already exist", len(existingDists2))
	} else {
		dt, ok := findType(10)
		if !ok {
			log.Printf("    Warning: distance type for 10K not found, skipping distances")
		} else {
			req := distances.CreateDistanceRequest{
				EventID:        evt2.ID,
				DistanceTypeID: dt.ID,
				Capacity:       1500,
				Surface:        "asphalt",
			}
			dist, err := distService.Create(ctx, req)
			if err != nil {
				log.Printf("    Warning: failed to create distance for Neon Dash: %v", err)
			} else {
				// Pricing stages for Neon Dash (regular only)
				existingStages, _ := s.store.Queries.ListPricingStagesByDistance(ctx, dist.ID)
				if len(existingStages) == 0 {
					basePrice := 4500.0
					if _, err := s.store.Queries.CreatePricingStage(ctx, db.CreatePricingStageParams{
						DistanceID:  dist.ID,
						Name:        pgtype.Text{String: "Regular", Valid: true},
						Price:       convert.ToPgNumeric(basePrice),
						StartDate:   neonDate.AddDate(0, -1, 0),
						EndDate:     neonDate,
						IsEarlyBird: pgtype.Bool{Bool: false, Valid: true},
					}); err != nil {
						log.Printf("    Warning: failed to create pricing for Neon Dash: %v", err)
					}
				}
			}
		}
	}

	// ─── EVENT 3: SUMMIT PEAK ─────────────────────────────────────────
	log.Println("  Setting up 'SUMMIT PEAK'...")

	org3, err := s.createOrGetOrganizer(ctx, "TRA240101ABC", "Trail Running Association",
		"Trail Running Association", "contact@trailrunning.com", "+1 555 987 6543",
		"Alpine Ridge", "Alpine", "80001")
	if err != nil {
		return fmt.Errorf("failed to setup Trail Running organizer: %w", err)
	}

	// Ensure 25K distance type exists (not in default seed)
	if _, ok := findType(25); !ok {
		log.Println("    25K distance type not found, creating inline...")
		newDt, err := s.store.Queries.CreateDistanceType(ctx, db.CreateDistanceTypeParams{
			Name: "25K Trail",
			Km:   convert.ToPgNumeric(25),
		})
		if err != nil {
			log.Printf("    Warning: failed to create 25K distance type: %v", err)
		} else {
			typeByKm["25.00"] = newDt
			distanceTypes = append(distanceTypes, newDt)
			log.Printf("    Created 25K distance type: %s", newDt.ID)
		}
	}

	summitDate := time.Date(2024, 12, 5, 5, 0, 0, 0, time.UTC)
	summitDeadline := time.Date(2024, 11, 20, 23, 59, 59, 0, time.UTC)

	evt3, err := s.createOrGetEvent(ctx, org3.ID, "SUMMIT PEAK", "summit-peak-trail",
		"Rugged mountain trail at sunset, athletic runner in distance, silhouette against orange sky, sharp peaks. A 25K trail running experience.",
		summitDate, summitDeadline,
		"Alpine Ridge", "Alpine", "Alpine Ridge Trail, CO",
		"https://lh3.googleusercontent.com/aida-public/AB6AXuD2qbDqeknWxsvvp_BJb5k4jzxOxnYjA0IhClvzMEcKYDSKyPHVR3G5es_vIALUERCkr2PCcrfQdhTAl_gTXve9vqZlQwNjCEdVHX1DYAMII9B1nydgNZ1Vfy4YTd2KVFBy_LI57nUvTUYnOwmC80iltHoTiG8h1Jfubz7z0A0XiiRu2a0bVNVrafntLb-zpbEGeLPM2sny-jG5j39S3x5SUhndIrI6e_GejNKkvoNCBGnNfYWBg5SFDlywDLHjv93z8xY7K1RXxHem",
		[]events.ItineraryItem{
			{Key: "Salida", Time: "05:00", Icon: "flag"},
			{Key: "Meta", Time: "10:00", Icon: "finish"},
		})
	if err != nil {
		return fmt.Errorf("failed to create Summit Peak event: %w", err)
	}
	s.createRunnerKits(ctx, evt3.ID)

	// Check if distances already exist
	existingDists3, _ := distService.ListByEvent(ctx, evt3.ID)
	if len(existingDists3) > 0 {
		log.Printf("    Skipping distances — %d already exist", len(existingDists3))
	} else {
		dt, ok := findType(25)
		if !ok {
			log.Printf("    Warning: 25K distance type not found, skipping")
		} else {
			req := distances.CreateDistanceRequest{
				EventID:        evt3.ID,
				DistanceTypeID: dt.ID,
				Capacity:       800,
				Surface:        "trail",
			}
			dist, err := distService.Create(ctx, req)
			if err != nil {
				log.Printf("    Warning: failed to create distance for Summit Peak: %v", err)
			} else {
				// Pricing — regular only (6000 USD)
				existingStages, _ := s.store.Queries.ListPricingStagesByDistance(ctx, dist.ID)
				if len(existingStages) == 0 {
					if _, err := s.store.Queries.CreatePricingStage(ctx, db.CreatePricingStageParams{
						DistanceID:  dist.ID,
						Name:        pgtype.Text{String: "Regular", Valid: true},
						Price:       convert.ToPgNumeric(6000.0),
						StartDate:   summitDate.AddDate(0, -1, 0),
						EndDate:     summitDate,
						IsEarlyBird: pgtype.Bool{Bool: false, Valid: true},
					}); err != nil {
						log.Printf("    Warning: failed to create pricing for Summit Peak: %v", err)
					}
				}
			}
		}
	}

	// ─── Auth users for mock organizers ──────────────────────────────
	log.Println("  Creating auth users for mock organizers...")

	defaultPassword := "seed1234"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashBytes)

	mockUsers := []struct {
		email       string
		organizerID uuid.UUID
	}{
		{"contact@valenciarunning.com", org1.ID},
		{"contact@nightrunning.com", org2.ID},
		{"contact@trailrunning.com", org3.ID},
	}

	for _, u := range mockUsers {
		_, lookupErr := s.store.Queries.GetUserByEmail(ctx, u.email)
		if lookupErr == nil {
			log.Printf("    Skipping user — already exists: %s", u.email)
			continue
		}

		params := db.CreateUserParams{
			Email:        u.email,
			PasswordHash: hashedPassword,
			Role:         "organizer",
			OrganizerID:  convert.ToPgUUID(u.organizerID),
		}

		if _, createErr := s.store.Queries.CreateUser(ctx, params); createErr != nil {
			log.Printf("    Warning: failed to create user %s: %v", u.email, createErr)
			continue
		}
		log.Printf("    Created user: %s / %s", u.email, defaultPassword)
	}

	return nil
}

// createRunnerKits creates sample runner kit items for a given event (idempotent)
func (s *Seeder) createRunnerKits(ctx context.Context, eventID uuid.UUID) {
	// Check if kit items already exist for this event
	existing, err := s.store.Queries.ListRunnerKitItemsByEvent(ctx, eventID)
	if err == nil && len(existing) > 0 {
		log.Printf("    Skipping runner kit items — %d already exist for event %s", len(existing), eventID)
		return
	}

	kitItems := []struct {
		name string
		icon string
	}{
		{name: "Playera Técnica DryFit", icon: "shirt"},
		{name: "Medalla Conmemorativa", icon: "medal"},
		{name: "Número de Corredor", icon: "timer"},
		{name: "Bolsa de Regalo", icon: "backpack"},
	}

	for _, k := range kitItems {
		_, err := s.store.Queries.CreateRunnerKitItem(ctx, db.CreateRunnerKitItemParams{
			EventID:     eventID,
			Name:        k.name,
			Icon:        k.icon,
		})
		if err != nil {
			log.Printf("    Warning: failed to create runner kit item '%s': %v", k.name, err)
			continue
		}
		log.Printf("    Created runner kit item: %s", k.name)
	}
}

// createOrGetOrganizer creates a new organizer or returns existing one
func (s *Seeder) createOrGetOrganizer(ctx context.Context, rfc, brandName, businessName, email, phone, state, city, zipCode string) (organizers.Organizer, error) {
	repo := organizers.NewRepository(s.store)
	service := organizers.NewService(repo, s.validator)
	existing, err := repo.GetByRFC(ctx, rfc)
	if err == nil && existing.ID != uuid.Nil {
		log.Printf("    Skipping organizer — already exists: %s", existing.BrandName.String)
		return existing, nil
	}

	req := organizers.CreateOrganizerRequest{
		BusinessName:   businessName,
		BrandName:      brandName,
		RFC:            rfc,
		BillingZipCode: zipCode,
		BillingState:   state,
		BillingCity:    city,
		ContactName:    brandName,
		ContactEmail:   email,
		ContactPhone:   phone,
		LogoUrl:        "",
	}
	org, err := service.Create(ctx, req)
	if err != nil {
		return organizers.Organizer{}, fmt.Errorf("failed to create organizer: %w", err)
	}
	log.Printf("    Created organizer: %s (%s)", org.BusinessName, org.ID)
	return org, nil
}

// createOrGetEvent creates a new event or returns existing one
func (s *Seeder) createOrGetEvent(ctx context.Context, organizerID uuid.UUID, name, slug, description string, eventDate, deadline time.Time, state, municipality, address, logoUrl string, itinerary []events.ItineraryItem) (events.Event, error) {
	repo := events.NewRepository(s.store)
	service := events.NewService(repo, s.validator)

	existing, err := repo.GetBySlug(ctx, slug)
	if err == nil && existing.ID != uuid.Nil {
		log.Printf("    Skipping event — already exists: %s", existing.Slug)
		return existing, nil
	}

	req := events.CreateEventRequest{
		OrganizerID:  organizerID,
		Name:         name,
		Slug:         slug,
		Description:  description,
		EventDate:    eventDate,
		Deadline:     deadline,
		State:        state,
		Municipality: municipality,
		Address:      address,
		LogoURL:      logoUrl,
		Itinerary:    itinerary,
	}
	evt, err := service.Create(ctx, req)
	if err != nil {
		return events.Event{}, fmt.Errorf("failed to create event '%s': %w", name, err)
	}
	log.Printf("    Created event: %s (%s)", evt.Name, evt.ID)
	return evt, nil
}

func (s *Seeder) createPricingStages(ctx context.Context, distances []db.Distance) error {
	repo := events.NewRepository(s.store)
	service := events.NewService(repo, s.validator)

	// Early bird pricing (60-90 days before event)
	earlyBirdStart := time.Now().AddDate(0, -2, 0)
	earlyBirdEnd := time.Now().AddDate(0, -1, 0)

	// Regular pricing (30-60 days before event)
	regStart := time.Now().AddDate(0, -1, 0)
	regEnd := time.Now().AddDate(0, 1, 0)

	// Late pricing (last 30 days)
	lateStart := time.Now().AddDate(0, 1, 0)
	lateEnd := time.Now().AddDate(0, 2, 0)

	for _, distance := range distances {
		// Check if pricing stages already exist for this distance
		existingStages, err := service.ListPricingStagesByDistance(ctx, distance.ID)
		if err == nil && len(existingStages) > 0 {
			log.Printf("  Skipping pricing stages — %d already exist for distance %s",
				len(existingStages), distance.ID)
			continue
		}

		basePrice := 350.0
		km := convert.PgNumericToFloat64(distance.Km)
		if km >= 21.1 {
			basePrice = 500.0
		}

		// Early bird stage
		earlyReq := events.CreatePricingStageRequest{
			DistanceID:  distance.ID,
			Name:        "Early Bird - Preventa",
			Price:       basePrice * 0.8, // 20% discount
			StartDate:   earlyBirdStart,
			EndDate:     earlyBirdEnd,
			IsEarlyBird: true,
		}
		if _, err := service.CreatePricingStage(ctx, earlyReq); err != nil {
			log.Printf("  Warning: failed to create early bird pricing: %v", err)
		}

		// Regular stage
		regReq := events.CreatePricingStageRequest{
			DistanceID:  distance.ID,
			Name:        "Precio Regular",
			Price:       basePrice,
			StartDate:   regStart,
			EndDate:     regEnd,
			IsEarlyBird: false,
		}
		if _, err := service.CreatePricingStage(ctx, regReq); err != nil {
			log.Printf("  Warning: failed to create regular pricing: %v", err)
		}

		// Late stage
		lateReq := events.CreatePricingStageRequest{
			DistanceID:  distance.ID,
			Name:        "Precio Tardío",
			Price:       basePrice * 1.3, // 30% surcharge
			StartDate:   lateStart,
			EndDate:     lateEnd,
			IsEarlyBird: false,
		}
		if _, err := service.CreatePricingStage(ctx, lateReq); err != nil {
			log.Printf("  Warning: failed to create late pricing: %v", err)
		}
	}

	return nil
}

