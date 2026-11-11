package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5/pgtype"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
)

// recorderRepo implements Repository and records every write. It never
// touches a database.
type recorderRepo struct {
	userByEmail       db.User
	getByEmailErr     error
	createOrganizer   []db.CreateOrganizerParams
	createParticipant []db.CreateParticipantParams
	users             []db.CreateUserParams
	createUserErr     error
	lastLoginCount    int
	getByID           db.User
}

func (r *recorderRepo) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if r.getByEmailErr != nil {
		return db.User{}, r.getByEmailErr
	}
	return r.userByEmail, nil
}

func (r *recorderRepo) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.getByID, nil
}

func (r *recorderRepo) CreateOrganizer(ctx context.Context, p db.CreateOrganizerParams) (db.Organizer, error) {
	r.createOrganizer = append(r.createOrganizer, p)
	o := db.Organizer{ID: uuid.New()}
	return o, nil
}

func (r *recorderRepo) CreateParticipant(ctx context.Context, p db.CreateParticipantParams) (db.Participant, error) {
	r.createParticipant = append(r.createParticipant, p)
	return db.Participant{ID: uuid.New()}, nil
}

func (r *recorderRepo) CreateUser(ctx context.Context, p db.CreateUserParams) (db.User, error) {
	if r.createUserErr != nil {
		return db.User{}, r.createUserErr
	}
	r.users = append(r.users, p)
	u := db.User{
		ID:       uuid.New(),
		Email:    p.Email,
		Role:     p.Role,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}
	return u, nil
}

func (r *recorderRepo) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) (db.User, error) {
	r.lastLoginCount++
	return r.getByID, nil
}

// merge commits the child recorder into the parent (tx semantics).
func (r *recorderRepo) merge(child *recorderRepo) {
	r.createOrganizer = append(r.createOrganizer, child.createOrganizer...)
	r.createParticipant = append(r.createParticipant, child.createParticipant...)
	r.users = append(r.users, child.users...)
	r.lastLoginCount += child.lastLoginCount
}

// committedRunner mimics Store.ExecTx: the forked repo's writes are merged
// into the parent only when fn succeeds (a failure rolls them back).
type committedRunner struct {
	order    *recorderRepo
	childErr error // error the tx-scoped child returns on CreateUser
	lastErr  error
}

func (f committedRunner) ExecTx(ctx context.Context, fn func(Repository) error) error {
	child := &recorderRepo{createUserErr: f.childErr}
	if err := fn(child); err != nil {
		f.lastErr = err
		return err
	}
	f.order.merge(child)
	return nil
}

func testService(repo Repository, tx TxRunner) *Service {
	return &Service{
		repo: repo,
		tx:   tx,
		cfg:  Config{SecretKey: "test-secret", TokenExpiry: time.Hour, RefreshExpiry: 24 * time.Hour},
	}
}

func hashedPassword(t *testing.T, raw string) string {
	t.Helper()
	hash, err := bcryptHash(raw)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func activeUser(t *testing.T, email string) db.User {
	return db.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword(t, "right-password"),
		Role:         "participant",
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
	}
}

func TestLogin_Success(t *testing.T) {
	u := activeUser(t, "runner@example.com")
	rec := &recorderRepo{userByEmail: u}
	svc := testService(rec, committedRunner{order: rec})

	tokens, claims, err := svc.Login(context.Background(), LoginRequest{Email: u.Email, Password: "right-password"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if claims.Email != u.Email || claims.Role != "participant" {
		t.Errorf("claims = %+v", claims)
	}
	got, err := ValidateAccessToken(svc.cfg, tokens.AccessToken)
	if err != nil || got.UserID != u.ID.String() {
		t.Errorf("access token invalid: %v", err)
	}
	if got, err := ValidateRefreshToken(svc.cfg, tokens.RefreshToken); err != nil || got.TokenType != TokenTypeRefresh {
		t.Errorf("refresh token invalid: %v", err)
	}
	if rec.lastLoginCount != 1 {
		t.Errorf("last login updates = %d, want 1", rec.lastLoginCount)
	}
}

func TestLogin_WrongPasswordDoesNotUpdateLastLogin(t *testing.T) {
	u := activeUser(t, "runner@example.com")
	rec := &recorderRepo{userByEmail: u}
	svc := testService(rec, committedRunner{order: rec})

	if _, _, err := svc.Login(context.Background(), LoginRequest{Email: u.Email, Password: "wrong"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
	if rec.lastLoginCount != 0 {
		t.Errorf("last login updates = %d, want 0", rec.lastLoginCount)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	u := activeUser(t, "runner@example.com")
	u.IsActive = pgtype.Bool{Bool: false, Valid: true}
	rec := &recorderRepo{userByEmail: u}
	svc := testService(rec, committedRunner{order: rec})

	if _, _, err := svc.Login(context.Background(), LoginRequest{Email: u.Email, Password: "right-password"}); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("err = %v, want ErrUserInactive", err)
	}
	if rec.lastLoginCount != 0 {
		t.Errorf("inactive user must not log in: updates = %d", rec.lastLoginCount)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	rec := &recorderRepo{getByEmailErr: errors.New("no rows")}
	svc := testService(rec, committedRunner{order: rec})

	if _, _, err := svc.Login(context.Background(), LoginRequest{Email: "ghost@example.com", Password: "x"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func dupEmailErr() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"users_email_key\""}
}

func organizerRequest() RegisterOrganizerRequest {
	return RegisterOrganizerRequest{
		Email:          "org@example.com",
		Password:       "long-enough-pass",
		BusinessName:   "Kineticos SA",
		BrandName:      "Kineticos",
		RFC:            "ABC123456XY1",
		BillingZipCode: "01000",
		BillingState:   "CDMX",
		BillingCity:    "CDMX",
		ContactName:    "Ana",
		ContactPhone:   "5550001111",
		LogoUrl:        "https://cdn.example/logo.png",
	}
}

func TestRegisterOrganizer_Success(t *testing.T) {
	rec := &recorderRepo{}
	svc := testService(rec, committedRunner{order: rec})

	res, err := svc.RegisterOrganizer(context.Background(), organizerRequest())
	if err != nil {
		t.Fatalf("RegisterOrganizer: %v", err)
	}
	if len(rec.createOrganizer) != 1 || len(rec.users) != 1 {
		t.Fatalf("writes = organizers %d users %d, want 1/1", len(rec.createOrganizer), len(rec.users))
	}
	if rec.users[0].Role != "organizer" || rec.users[0].Email != "org@example.com" {
		t.Errorf("user = %+v", rec.users[0])
	}
	if len(rec.users[0].PasswordHash) < 20 || strings.Contains(rec.users[0].PasswordHash, "long-enough-pass") {
		t.Error("password must be bcrypt-hashed")
	}
	if res.AccessToken == "" || res.RefreshToken == "" || res.ExpiresIn != 3600 {
		t.Errorf("tokens = %+v", res)
	}
}

func TestRegisterOrganizer_DuplicateEmail(t *testing.T) {
	rec := &recorderRepo{userByEmail: activeUser(t, "org@example.com")}
	runner := committedRunner{order: rec, childErr: dupEmailErr()}
	svc := testService(rec, runner)

	if _, err := svc.RegisterOrganizer(context.Background(), organizerRequest()); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v (lastErr=%v), want ErrEmailTaken", err, runner.lastErr)
	}
	if len(rec.createOrganizer) != 0 || len(rec.users) != 0 {
		t.Errorf("duplicate email must not write: organizers %d users %d", len(rec.createOrganizer), len(rec.users))
	}
}

func TestRegisterOrganizer_UserConflictRollsBackOrganizer(t *testing.T) {
	rec := &recorderRepo{createUserErr: dupEmailErr()}
	runner := committedRunner{order: rec, childErr: dupEmailErr()}
	svc := testService(rec, runner)

	if _, err := svc.RegisterOrganizer(context.Background(), organizerRequest()); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v (lastErr=%v), want ErrEmailTaken", err, runner.lastErr)
	}
	if len(rec.createOrganizer) != 0 || len(rec.users) != 0 {
		t.Errorf("failed tx must not persist: organizers %d users %d", len(rec.createOrganizer), len(rec.users))
	}
}

func participantRequest() RegisterParticipantRequest {
	return RegisterParticipantRequest{
		Email:          "runner@example.com",
		Password:       "long-enough-pass",
		FirstName:      "Pedro",
		LastName:       "Martinez",
		SecondLastName: pmString("Alvarez"),
		BirthDate:      time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC),
		Gender:         "male",
		Phone:          "5551234567",
	}
}

func pmString(s string) *string { return &s }

func TestRegisterParticipant_Success(t *testing.T) {
	rec := &recorderRepo{}
	svc := testService(rec, committedRunner{order: rec})

	res, err := svc.RegisterParticipant(context.Background(), participantRequest())
	if err != nil {
		t.Fatalf("RegisterParticipant: %v", err)
	}
	p := rec.createParticipant[0]
	if p.FirstName != "Pedro" || p.Email != "runner@example.com" {
		t.Errorf("participant = %+v", p)
	}
	if len(rec.users) != 1 || rec.users[0].Role != "participant" {
		t.Errorf("users = %+v", rec.users)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Errorf("tokens missing: %+v", res)
	}
}

func TestRegisterParticipant_DuplicateEmail(t *testing.T) {
	rec := &recorderRepo{userByEmail: activeUser(t, "runner@example.com")}
	svc := testService(rec, committedRunner{order: rec})

	if _, err := svc.RegisterParticipant(context.Background(), participantRequest()); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
	if len(rec.createParticipant) != 0 || len(rec.users) != 0 {
		t.Errorf("duplicate email must not write: participants %d users %d", len(rec.createParticipant), len(rec.users))
	}
}

func TestRegisterParticipant_UserConflictRollsBackParticipant(t *testing.T) {
	rec := &recorderRepo{}
	svc := testService(rec, committedRunner{order: rec, childErr: dupEmailErr()})

	if _, err := svc.RegisterParticipant(context.Background(), participantRequest()); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
	if len(rec.createParticipant) != 0 || len(rec.users) != 0 {
		t.Errorf("failed tx must not persist: participants %d users %d", len(rec.createParticipant), len(rec.users))
	}
}
