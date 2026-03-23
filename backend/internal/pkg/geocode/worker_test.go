package geocode

import (
	"context"
	"errors"
	"testing"
	"time"

	"chasing_points/internal/model"
)

type fakeVenueRepository struct {
	venue           *model.Venue
	findErr         error
	successVenueID  int64
	successAttempts int
	successSource   string
	failureVenueID  int64
	failureAttempts int
	failureStatus   int
	failureMessage  string
}

func (f *fakeVenueRepository) FindById(id int64) (*model.Venue, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.venue != nil && f.venue.Id == id {
		return f.venue, nil
	}
	return nil, nil
}

func (f *fakeVenueRepository) ApplyGeocodeResult(venueId int64, result *Result, attempts int, autoPublish bool) error {
	f.successVenueID = venueId
	f.successAttempts = attempts
	f.successSource = result.Source
	return nil
}

func (f *fakeVenueRepository) MarkGeocodeFailure(venueId int64, geoStatus int, attempts int, message string) error {
	f.failureVenueID = venueId
	f.failureStatus = geoStatus
	f.failureAttempts = attempts
	f.failureMessage = message
	return nil
}

type fakeTaskRepository struct {
	successIDs    []int64
	retryTaskID   int64
	retryAttempts int
	retryAt       time.Time
	retryMessage  string
	deadTaskID    int64
	deadAttempts  int
	deadMessage   string
	releaseTaskID int64
	releaseAt     time.Time
	releaseMsg    string
}

func (f *fakeTaskRepository) MarkSuccess(taskId int64) error {
	f.successIDs = append(f.successIDs, taskId)
	return nil
}

func (f *fakeTaskRepository) MarkRetry(taskId int64, attempts int, nextRetryAt time.Time, message string) error {
	f.retryTaskID = taskId
	f.retryAttempts = attempts
	f.retryAt = nextRetryAt
	f.retryMessage = message
	return nil
}

func (f *fakeTaskRepository) MarkDead(taskId int64, attempts int, message string) error {
	f.deadTaskID = taskId
	f.deadAttempts = attempts
	f.deadMessage = message
	return nil
}

func (f *fakeTaskRepository) ReleaseClaim(taskId int64, nextRetryAt time.Time, message string) error {
	f.releaseTaskID = taskId
	f.releaseAt = nextRetryAt
	f.releaseMsg = message
	return nil
}

type fakeAccountRepository struct {
	accounts     []model.GeocodeAccount
	disabledIDs  []int64
	successIDs   []int64
	cooldownID   int64
	cooldownTime time.Time
}

func (f *fakeAccountRepository) ListEnabled(_ string, _ time.Time) ([]model.GeocodeAccount, error) {
	return f.accounts, nil
}

func (f *fakeAccountRepository) UpdateCooldown(accountId int64, coolDownUntil time.Time) error {
	f.cooldownID = accountId
	f.cooldownTime = coolDownUntil
	return nil
}

func (f *fakeAccountRepository) TouchSuccess(accountId int64) error {
	f.successIDs = append(f.successIDs, accountId)
	return nil
}

func (f *fakeAccountRepository) MarkDisabled(accountId int64) error {
	f.disabledIDs = append(f.disabledIDs, accountId)
	return nil
}

type fakeAccountSelector struct {
	account *model.GeocodeAccount
	err     error
}

func (f *fakeAccountSelector) Select(_ context.Context, _ []model.GeocodeAccount, _ time.Time) (*model.GeocodeAccount, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.account, nil
}

type fakeGeocoder struct {
	result *Result
	err    error
}

func (f *fakeGeocoder) Geocode(_ context.Context, _ model.GeocodeAccount, _ string) (*Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeTaskQueue struct {
	tasks       []model.VenueGeocodeTask
	claimResult bool
	claimErr    error
	claimedIDs  []int64
}

func (f *fakeTaskQueue) ListDueTasks(_ int, _ time.Time) ([]model.VenueGeocodeTask, error) {
	return f.tasks, nil
}

func (f *fakeTaskQueue) ClaimTask(taskId int64, _ string, _ time.Time) (bool, error) {
	if f.claimErr != nil {
		return false, f.claimErr
	}
	f.claimedIDs = append(f.claimedIDs, taskId)
	return f.claimResult, nil
}

func TestWorkerProcessTaskPublishesVenueOnSuccess(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 0, 0, time.UTC)
	worker := &Worker{
		venueRepo: &fakeVenueRepository{
			venue: &model.Venue{Id: 9, FullAddress: "上海市浦东新区东明路"},
		},
		taskRepo: &fakeTaskRepository{},
		accountRepo: &fakeAccountRepository{
			accounts: []model.GeocodeAccount{{Id: 7, Status: model.GeocodeAccountStatusEnabled, Provider: "apihz"}},
		},
		selector: &fakeAccountSelector{
			account: &model.GeocodeAccount{Id: 7, Status: model.GeocodeAccountStatusEnabled, Provider: "apihz"},
		},
		geocoder: &fakeGeocoder{
			result: &Result{Longitude: 121.496, Latitude: 31.14238, Score: 75, Level: "乡镇街道", Source: "apihz"},
		},
		autoPublish: true,
	}

	task := model.VenueGeocodeTask{Id: 3, VenueId: 9, Attempts: 1, MaxAttempts: 5}
	if err := worker.processTask(context.Background(), task, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	venueRepo := worker.venueRepo.(*fakeVenueRepository)
	taskRepo := worker.taskRepo.(*fakeTaskRepository)
	accountRepo := worker.accountRepo.(*fakeAccountRepository)

	if venueRepo.successVenueID != 9 {
		t.Fatalf("expected venue 9 to be updated, got %d", venueRepo.successVenueID)
	}
	if venueRepo.successAttempts != 2 {
		t.Fatalf("expected attempts 2, got %d", venueRepo.successAttempts)
	}
	if len(taskRepo.successIDs) != 1 || taskRepo.successIDs[0] != 3 {
		t.Fatalf("expected task 3 success, got %+v", taskRepo.successIDs)
	}
	if len(accountRepo.successIDs) != 1 || accountRepo.successIDs[0] != 7 {
		t.Fatalf("expected account 7 success touch, got %+v", accountRepo.successIDs)
	}
}

func TestWorkerProcessTaskSchedulesRetryWhenNoAccountAvailable(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 20, 0, time.UTC)
	worker := &Worker{
		venueRepo: &fakeVenueRepository{
			venue: &model.Venue{Id: 9, FullAddress: "上海市浦东新区东明路"},
		},
		taskRepo: &fakeTaskRepository{},
		accountRepo: &fakeAccountRepository{
			accounts: []model.GeocodeAccount{{Id: 7, Status: model.GeocodeAccountStatusEnabled, Provider: "apihz"}},
		},
		selector: &fakeAccountSelector{err: ErrNoAvailableAccount},
		geocoder: &fakeGeocoder{},
	}

	task := model.VenueGeocodeTask{Id: 3, VenueId: 9, Attempts: 0, MaxAttempts: 5}
	if err := worker.processTask(context.Background(), task, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	taskRepo := worker.taskRepo.(*fakeTaskRepository)
	venueRepo := worker.venueRepo.(*fakeVenueRepository)
	if taskRepo.retryTaskID != 3 {
		t.Fatalf("expected task 3 retry, got %d", taskRepo.retryTaskID)
	}
	if taskRepo.retryAttempts != 0 {
		t.Fatalf("expected attempts to stay 0, got %d", taskRepo.retryAttempts)
	}
	if venueRepo.failureVenueID != 0 {
		t.Fatalf("expected venue geocode failure not to be recorded, got venue %d", venueRepo.failureVenueID)
	}
	expectedRetryAt := time.Date(2026, 3, 14, 9, 16, 2, 0, time.UTC)
	if !taskRepo.retryAt.Equal(expectedRetryAt) {
		t.Fatalf("expected retry at %v, got %v", expectedRetryAt, taskRepo.retryAt)
	}
}

func TestWorkerProcessTaskDisablesAccountOnAuthFailure(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 20, 0, time.UTC)
	worker := &Worker{
		venueRepo: &fakeVenueRepository{
			venue: &model.Venue{Id: 9, FullAddress: "上海市浦东新区东明路"},
		},
		taskRepo: &fakeTaskRepository{},
		accountRepo: &fakeAccountRepository{
			accounts: []model.GeocodeAccount{{Id: 7, Status: model.GeocodeAccountStatusEnabled, Provider: "apihz"}},
		},
		selector: &fakeAccountSelector{
			account: &model.GeocodeAccount{Id: 7, Status: model.GeocodeAccountStatusEnabled, Provider: "apihz"},
		},
		geocoder: &fakeGeocoder{
			err: NewProviderError(ErrorKindAuth, "key invalid"),
		},
	}

	task := model.VenueGeocodeTask{Id: 3, VenueId: 9, Attempts: 1, MaxAttempts: 5}
	if err := worker.processTask(context.Background(), task, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	taskRepo := worker.taskRepo.(*fakeTaskRepository)
	accountRepo := worker.accountRepo.(*fakeAccountRepository)
	venueRepo := worker.venueRepo.(*fakeVenueRepository)

	if len(accountRepo.disabledIDs) != 1 || accountRepo.disabledIDs[0] != 7 {
		t.Fatalf("expected account 7 to be disabled, got %+v", accountRepo.disabledIDs)
	}
	if taskRepo.retryTaskID != 3 {
		t.Fatalf("expected task 3 retry, got %d", taskRepo.retryTaskID)
	}
	if venueRepo.failureStatus != model.VenueGeoStatusRetrying {
		t.Fatalf("expected venue retrying status, got %d", venueRepo.failureStatus)
	}
}

func TestWorkerProcessTaskMarksDeadWhenVenueMissing(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 20, 0, time.UTC)
	worker := &Worker{
		venueRepo:   &fakeVenueRepository{},
		taskRepo:    &fakeTaskRepository{},
		accountRepo: &fakeAccountRepository{},
		selector:    &fakeAccountSelector{},
		geocoder:    &fakeGeocoder{},
	}

	task := model.VenueGeocodeTask{Id: 3, VenueId: 9, Attempts: 4, MaxAttempts: 5}
	if err := worker.processTask(context.Background(), task, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	taskRepo := worker.taskRepo.(*fakeTaskRepository)
	if taskRepo.deadTaskID != 3 {
		t.Fatalf("expected task 3 dead, got %d", taskRepo.deadTaskID)
	}
}

func TestWorkerRunOnceReleasesClaimWhenProcessTaskReturnsError(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 20, 0, time.UTC)
	queue := &fakeTaskQueue{
		tasks:       []model.VenueGeocodeTask{{Id: 3, VenueId: 9, Attempts: 1, MaxAttempts: 5}},
		claimResult: true,
	}
	taskRepo := &fakeTaskRepository{}
	worker := &Worker{
		venueRepo: &fakeVenueRepository{
			findErr: errors.New("db temporarily unavailable"),
		},
		taskQueue: queue,
		taskRepo:  taskRepo,
	}

	err := worker.RunOnce(context.Background(), now)
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	if taskRepo.releaseTaskID != 3 {
		t.Fatalf("expected task 3 claim released, got %d", taskRepo.releaseTaskID)
	}
	expectedRetryAt := time.Date(2026, 3, 14, 9, 16, 2, 0, time.UTC)
	if !taskRepo.releaseAt.Equal(expectedRetryAt) {
		t.Fatalf("expected release retry at %v, got %v", expectedRetryAt, taskRepo.releaseAt)
	}
}

func TestWorkerProcessTaskReturnsErrorWhenVenueLookupFails(t *testing.T) {
	now := time.Date(2026, 3, 14, 9, 15, 20, 0, time.UTC)
	worker := &Worker{
		venueRepo: &fakeVenueRepository{
			findErr: errors.New("db down"),
		},
		taskRepo:    &fakeTaskRepository{},
		accountRepo: &fakeAccountRepository{},
		selector:    &fakeAccountSelector{},
		geocoder:    &fakeGeocoder{},
	}

	task := model.VenueGeocodeTask{Id: 3, VenueId: 9, Attempts: 0, MaxAttempts: 5}
	if err := worker.processTask(context.Background(), task, now); err == nil {
		t.Fatal("expected error, got nil")
	}
}
