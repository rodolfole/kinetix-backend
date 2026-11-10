package registrations

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// fakeRepo implements ServiceRepository and records writes.
type fakeRepo struct {
	existingRegistration Registration
	existingErr          error
	created              []CreateRegistrationRequest
	createErr            error

	participants      []CreateParticipantData
	getParticipant    repo.Participant
	getParticipantErr error
	updates           []repo.UpdateParticipantParams
	updateErr         error

	fields    []repo.EventParticipantField
	fieldsErr error

	deleted    []uuid.UUID
	deletedErr error
}

func (f *fakeRepo) Create(ctx context.Context, req CreateRegistrationRequest) (Registration, error) {
	if f.createErr != nil {
		return Registration{}, f.createErr
	}
	f.created = append(f.created, req)
	r := Registration{ID: uuid.New(), EventID: req.EventID, ParticipantID: req.ParticipantID, DistanceID: req.DistanceID}
	return r, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id uuid.UUID) (Registration, error) {
	return Registration{ID: id}, nil
}

func (f *fakeRepo) GetByEventParticipant(ctx context.Context, eventID, participantID uuid.UUID) (Registration, error) {
	if f.existingErr != nil {
		return Registration{}, f.existingErr
	}
	return f.existingRegistration, nil
}

func (f *fakeRepo) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Registration, error) {
	return nil, nil
}

func (f *fakeRepo) ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Registration, error) {
	return nil, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if f.deletedErr != nil {
		return f.deletedErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeRepo) CreateParticipant(ctx context.Context, p CreateParticipantData) (Participant, error) {
	f.participants = append(f.participants, p)
	return Participant{ID: uuid.New(), Email: p.Email}, nil
}

func (f *fakeRepo) GetParticipant(ctx context.Context, id uuid.UUID) (repo.Participant, error) {
	if f.getParticipantErr != nil {
		return repo.Participant{}, f.getParticipantErr
	}
	return f.getParticipant, nil
}

func (f *fakeRepo) UpdateParticipant(ctx context.Context, p repo.UpdateParticipantParams) (repo.Participant, error) {
	if f.updateErr != nil {
		return repo.Participant{}, f.updateErr
	}
	f.updates = append(f.updates, p)
	return f.getParticipant, nil
}

func (f *fakeRepo) EventParticipantFields(ctx context.Context, eventID uuid.UUID) ([]repo.EventParticipantField, error) {
	if f.fieldsErr != nil {
		return nil, f.fieldsErr
	}
	return f.fields, nil
}

// fakeTx mimics a transaction: the child sees the parent's config; writes are
// merged into the parent on success, discarded on error. createErrFor maps
// each attempt to a Create error deterministically.
type fakeTx struct {
	parent       *fakeRepo
	createErrFor func(attempt int) error
	updateErr    error
	invoked      int
	merged       bool
}

func (f *fakeTx) ExecTx(ctx context.Context, fn func(ServiceRepository) error) error {
	f.invoked++
	attempt := f.invoked - 1
	var createErr error
	if f.createErrFor != nil {
		createErr = f.createErrFor(attempt)
	}
	child := &fakeRepo{
		existingRegistration: f.parent.existingRegistration,
		existingErr:          f.parent.existingErr,
		createErr:            createErr,
		getParticipant:       f.parent.getParticipant,
		getParticipantErr:    f.parent.getParticipantErr,
		updateErr:            f.updateErr,
		fields:               f.parent.fields,
	}
	if err := fn(child); err != nil {
		return err
	}
	// commit: merge writes into parent
	f.parent.created = append(f.parent.created, child.created...)
	f.parent.participants = append(f.parent.participants, child.participants...)
	f.parent.updates = append(f.parent.updates, child.updates...)
	f.merged = true
	return nil
}

func testSvc(repo ServiceRepository, tx TxRunner) *Service {
	return NewService(repo, tx, validator.New())
}

func validRegReq() CreateRegistrationRequest {
	return CreateRegistrationRequest{
		EventID:       uuid.New(),
		ParticipantID: uuid.New(),
		DistanceID:    uuid.New(),
		Category:      "general",
	}
}

func TestCreate_DuplicateParticipant(t *testing.T) {
	repo := &fakeRepo{existingRegistration: Registration{ID: uuid.New()}}
	svc := testSvc(repo, &fakeTx{parent: repo})

	if _, err := svc.Create(context.Background(), validRegReq()); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("err = %v, want ErrAlreadyRegistered", err)
	}
	if len(repo.created) != 0 {
		t.Errorf("duplicates must not write, created = %d", len(repo.created))
	}
}

func TestCreate_UsesTransactionOnSuccess(t *testing.T) {
	repo := &fakeRepo{}
	tx := &fakeTx{parent: repo}
	svc := testSvc(repo, tx)

	got, err := svc.Create(context.Background(), validRegReq())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.EventID != repo.created[0].EventID {
		t.Errorf("returned registration %v != created %v", got.EventID, repo.created[0].EventID)
	}
	if tx.invoked != 1 {
		t.Errorf("tx invoked %d times, want 1", tx.invoked)
	}
	if !tx.merged {
		t.Error("tx writes must be committed (merged into parents)")
	}
}

func TestCreate_BibConflictRetries(t *testing.T) {
	repo := &fakeRepo{}
	bibErr := &pgconn.PgError{Code: "23505", ConstraintName: "registrations_bib_number_key", Message: "duplicate key"}
	tx := &fakeTx{parent: repo, createErrFor: func(a int) error {
		if a < 2 {
			return bibErr
		}
		return nil
	}}
	svc := testSvc(repo, tx)

	if _, err := svc.Create(context.Background(), validRegReq()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tx.invoked != 3 {
		t.Errorf("attempts = %d, want 3 (two bib conflicts + success)", tx.invoked)
	}
}

func TestCreate_BibConflictExhausted(t *testing.T) {
	repo := &fakeRepo{}
	bibErr := &pgconn.PgError{Code: "23505", ConstraintName: "registrations_bib_number_key", Message: "duplicate key"}
	tx := &fakeTx{parent: repo, createErrFor: func(int) error { return bibErr }}
	svc := testSvc(repo, tx)

	_, err := svc.Create(context.Background(), validRegReq())
	if !errors.Is(err, errBibConflict) {
		t.Fatalf("err = %v, want errBibConflict after retries", err)
	}
	if len(repo.created) != 0 {
		t.Errorf("failed attempts must not persist, created = %d", len(repo.created))
	}
}

func TestCreate_NonBibConflictDoesNotRetry(t *testing.T) {
	repo := &fakeRepo{}
	otherErr := &pgconn.PgError{Code: "23505", ConstraintName: "registrations_event_participant_distance_key", Message: "duplicate"}
	tx := &fakeTx{parent: repo, createErrFor: func(int) error { return otherErr }}
	svc := testSvc(repo, tx)

	if _, err := svc.Create(context.Background(), validRegReq()); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v, want the original unique error", err)
	}
	if tx.invoked != 1 {
		t.Errorf("non-bib unique conflicts retry? invoked = %d, want 1", tx.invoked)
	}
}

func TestCreateWithParticipant_ExistingParticipantInTx(t *testing.T) {
	repo := &fakeRepo{getParticipant: repo.Participant{ID: uuid.New(), FirstName: "Pedro"}}
	tx := &fakeTx{parent: repo}
	svc := testSvc(repo, tx)

	pid := repo.getParticipant.ID
	data := CreateParticipantData{FirstName: "Pedro"}
	regData := CreateWithParticipantRegistration{
		DistanceID: uuid.New(),
		Category:   "general",
		Team:       teamPtr("Kolores"),
	}

	got, err := svc.CreateWithParticipant(context.Background(), uuid.New(), data, regData, &pid)
	if err != nil {
		t.Fatalf("CreateWithParticipant: %v", err)
	}
	if len(repo.updates) != 1 || repo.updates[0].ID != pid {
		t.Errorf("participant updates = %+v, want one for %s", repo.updates, pid)
	}
	if len(repo.created) != 1 || repo.created[0].ParticipantID != pid {
		t.Errorf("registrations = %+v, want one for the existing participant", repo.created)
	}
	_ = got
}

func TestCreateWithParticipant_NewParticipantInTx(t *testing.T) {
	repo := &fakeRepo{}
	tx := &fakeTx{parent: repo}
	svc := testSvc(repo, tx)

	eventID := uuid.New()
	got, err := svc.CreateWithParticipant(context.Background(), eventID, CreateParticipantData{FirstName: "Ana", Email: "ana@x.com"}, CreateWithParticipantRegistration{DistanceID: uuid.New(), Category: "general"}, nil)
	if err != nil {
		t.Fatalf("CreateWithParticipant: %v", err)
	}
	if len(repo.participants) != 1 {
		t.Fatalf("participants = %d, want 1", len(repo.participants))
	}
	if repo.created[0].ParticipantID == uuid.Nil {
		t.Errorf("registration must link the created participant: %+v", repo.created[0])
	}
	if got.ParticipantID == uuid.Nil {
		t.Errorf("returned registration lacks the participant: %+v", got)
	}
}

func TestCreateWithParticipant_EnforcesRequiredFields(t *testing.T) {
	repo := &fakeRepo{fields: []repo.EventParticipantField{
		{FieldName: "country", IsEnabled: true, IsRequired: true},
	}}
	svc := testSvc(repo, &fakeTx{parent: repo})

	if _, err := svc.CreateWithParticipant(context.Background(), uuid.New(), CreateParticipantData{}, CreateWithParticipantRegistration{DistanceID: uuid.New(), Category: "general"}, nil); err == nil {
		t.Fatal("missing required country must fail")
	}
	if len(repo.created) != 0 || len(repo.participants) != 0 {
		t.Error("validation failure must not write anything")
	}
}

func TestCreateWithParticipant_UpdateFailureRollsBackRegistration(t *testing.T) {
	updateErr := &pgconn.PgError{Code: "40001", Message: "could not serialize due to concurrent update"}
	repo := &fakeRepo{getParticipant: repo.Participant{ID: uuid.New(), FirstName: "Pedro"}}
	tx := &fakeTx{parent: repo, updateErr: updateErr}
	svc := testSvc(repo, tx)

	pid := repo.getParticipant.ID
	if _, err := svc.CreateWithParticipant(context.Background(), uuid.New(), CreateParticipantData{}, CreateWithParticipantRegistration{DistanceID: uuid.New(), Category: "general"}, &pid); err == nil {
		t.Fatal("failing participant update must fail the flow")
	}
	if len(repo.created) != 0 {
		t.Errorf(" rolled-back tx must not leave a registration, got %d", len(repo.created))
	}
	if len(repo.updates) != 0 {
		t.Errorf("rolled-back tx must not keep the participant update, got %d", len(repo.updates))
	}
}

// -- helpers

func teamPtr(s string) *string { return &s }
