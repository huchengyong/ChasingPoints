package geocode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chasing_points/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type VenueRepository interface {
	FindById(id int64) (*model.Venue, error)
	ApplyGeocodeResult(venueId int64, result *Result, attempts int, autoPublish bool) error
	MarkGeocodeFailure(venueId int64, geoStatus int, attempts int, message string) error
}

type DueTaskRepository interface {
	ListDueTasks(limit int, now time.Time) ([]model.VenueGeocodeTask, error)
	ClaimTask(taskId int64, locker string, now time.Time) (bool, error)
}

type TaskStateRepository interface {
	MarkSuccess(taskId int64) error
	MarkRetry(taskId int64, attempts int, nextRetryAt time.Time, message string) error
	MarkDead(taskId int64, attempts int, message string) error
	ReleaseClaim(taskId int64, nextRetryAt time.Time, message string) error
}

type AccountRepository interface {
	ListEnabled(provider string, now time.Time) ([]model.GeocodeAccount, error)
	UpdateCooldown(accountId int64, coolDownUntil time.Time) error
	TouchSuccess(accountId int64) error
	MarkDisabled(accountId int64) error
}

type Selector interface {
	Select(ctx context.Context, accounts []model.GeocodeAccount, now time.Time) (*model.GeocodeAccount, error)
}

type Worker struct {
	logger       logx.Logger
	venueRepo    VenueRepository
	taskQueue    DueTaskRepository
	taskRepo     TaskStateRepository
	accountRepo  AccountRepository
	selector     Selector
	geocoder     Geocoder
	provider     string
	lockerID     string
	batchSize    int
	pollInterval time.Duration
	autoPublish  bool
}

func NewWorker(
	venueRepo VenueRepository,
	taskQueue DueTaskRepository,
	taskRepo TaskStateRepository,
	accountRepo AccountRepository,
	selector Selector,
	geocoder Geocoder,
	provider, lockerID string,
	batchSize int,
	pollInterval time.Duration,
	autoPublish bool,
) *Worker {
	if provider == "" {
		provider = "apihz"
	}
	if lockerID == "" {
		lockerID = "chasing_points-api"
	}
	if batchSize <= 0 {
		batchSize = 5
	}
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}

	return &Worker{
		logger:       logx.WithContext(context.Background()),
		venueRepo:    venueRepo,
		taskQueue:    taskQueue,
		taskRepo:     taskRepo,
		accountRepo:  accountRepo,
		selector:     selector,
		geocoder:     geocoder,
		provider:     provider,
		lockerID:     lockerID,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		autoPublish:  autoPublish,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		if err := w.RunOnce(ctx, time.Now()); err != nil {
			w.logger.Errorf("geocode worker run once failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context, now time.Time) error {
	if w.taskQueue == nil {
		return nil
	}

	tasks, err := w.taskQueue.ListDueTasks(w.batchSize, now)
	if err != nil {
		return err
	}

	var runErr error
	for _, task := range tasks {
		claimed, err := w.taskQueue.ClaimTask(task.Id, w.lockerID, now)
		if err != nil {
			if runErr == nil {
				runErr = err
			}
			continue
		}
		if !claimed {
			continue
		}
		if err := w.processTask(ctx, task, now); err != nil {
			releaseErr := w.taskRepo.ReleaseClaim(task.Id, nextMinuteRetryAt(now), err.Error())
			if releaseErr != nil {
				err = errors.Join(err, releaseErr)
			}
			if runErr == nil {
				runErr = err
			} else {
				runErr = errors.Join(runErr, err)
			}
		}
	}

	return runErr
}

func (w *Worker) processTask(ctx context.Context, task model.VenueGeocodeTask, now time.Time) error {
	venue, err := w.venueRepo.FindById(task.VenueId)
	if err != nil {
		return err
	}
	if venue == nil {
		return w.taskRepo.MarkDead(task.Id, task.Attempts, "venue not found")
	}

	address := venue.FullAddress
	if address == "" {
		address = model.BuildVenueFullAddress(venue.City, venue.District, venue.Address)
	}

	accounts, err := w.accountRepo.ListEnabled(w.provider, now)
	if err != nil {
		return err
	}

	account, err := w.selector.Select(ctx, accounts, now)
	if err != nil {
		if errors.Is(err, ErrNoAvailableAccount) {
			return w.taskRepo.MarkRetry(task.Id, task.Attempts, nextMinuteRetryAt(now), err.Error())
		}
		return err
	}

	result, err := w.geocoder.Geocode(ctx, *account, address)
	if err != nil {
		return w.handleGeocodeError(task, *account, err, now)
	}

	attempts := task.Attempts + 1
	if err := w.venueRepo.ApplyGeocodeResult(task.VenueId, result, attempts, w.autoPublish); err != nil {
		return err
	}
	if err := w.taskRepo.MarkSuccess(task.Id); err != nil {
		return err
	}
	return w.accountRepo.TouchSuccess(account.Id)
}

func (w *Worker) handleGeocodeError(task model.VenueGeocodeTask, account model.GeocodeAccount, err error, now time.Time) error {
	attempts := task.Attempts + 1
	providerErr := &ProviderError{Kind: ErrorKindTemporary}
	if errors.As(err, &providerErr) {
		switch providerErr.Kind {
		case ErrorKindAuth:
			if err := w.accountRepo.MarkDisabled(account.Id); err != nil {
				return err
			}
			return w.retryOrDead(task, attempts, providerErr.Message, now, model.VenueGeoStatusRetrying, nextMinuteRetryAt(now))
		case ErrorKindTemporary:
			if err := w.accountRepo.UpdateCooldown(account.Id, now.Add(time.Minute)); err != nil {
				return err
			}
			return w.retryOrDead(task, attempts, providerErr.Message, now, model.VenueGeoStatusRetrying, retryAtForAttempt(now, attempts))
		case ErrorKindInvalidResponse:
			return w.retryOrDead(task, attempts, providerErr.Message, now, model.VenueGeoStatusRetrying, retryAtForAttempt(now, attempts))
		}
	}

	return w.retryOrDead(task, attempts, err.Error(), now, model.VenueGeoStatusRetrying, retryAtForAttempt(now, attempts))
}

func (w *Worker) retryOrDead(task model.VenueGeocodeTask, attempts int, message string, now time.Time, retryStatus int, retryAt time.Time) error {
	if attempts >= maxAttempts(task.MaxAttempts) {
		if err := w.venueRepo.MarkGeocodeFailure(task.VenueId, model.VenueGeoStatusFailed, attempts, message); err != nil {
			return err
		}
		return w.taskRepo.MarkDead(task.Id, attempts, message)
	}

	if err := w.venueRepo.MarkGeocodeFailure(task.VenueId, retryStatus, attempts, message); err != nil {
		return err
	}
	return w.taskRepo.MarkRetry(task.Id, attempts, retryAt, message)
}

func maxAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return 5
	}
	return maxAttempts
}

func nextMinuteRetryAt(now time.Time) time.Time {
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	return nextMinute.Add(2 * time.Second)
}

func retryAtForAttempt(now time.Time, attempts int) time.Time {
	switch {
	case attempts <= 1:
		return now.Add(1 * time.Minute)
	case attempts == 2:
		return now.Add(3 * time.Minute)
	case attempts == 3:
		return now.Add(10 * time.Minute)
	default:
		return now.Add(30 * time.Minute)
	}
}

type venueModelRepo struct {
	model *model.VenueModel
}

func NewVenueModelRepo(m *model.VenueModel) VenueRepository {
	return &venueModelRepo{model: m}
}

func (r *venueModelRepo) FindById(id int64) (*model.Venue, error) {
	return r.model.FindById(id)
}

func (r *venueModelRepo) ApplyGeocodeResult(venueId int64, result *Result, attempts int, autoPublish bool) error {
	return r.model.ApplyGeocodeResultWithTx(nil, venueId, result.Longitude, result.Latitude, result.Score, result.Level, result.Source, attempts, autoPublish)
}

func (r *venueModelRepo) MarkGeocodeFailure(venueId int64, geoStatus int, attempts int, message string) error {
	return r.model.MarkGeocodeFailureWithTx(nil, venueId, geoStatus, attempts, message)
}

type taskModelRepo struct {
	model *model.VenueGeocodeTaskModel
}

func NewTaskQueueRepo(m *model.VenueGeocodeTaskModel) DueTaskRepository {
	return &taskModelRepo{model: m}
}

func NewTaskStateRepo(m *model.VenueGeocodeTaskModel) TaskStateRepository {
	return &taskModelRepo{model: m}
}

func (r *taskModelRepo) ListDueTasks(limit int, now time.Time) ([]model.VenueGeocodeTask, error) {
	return r.model.ListDueTasks(limit, now)
}

func (r *taskModelRepo) ClaimTask(taskId int64, locker string, now time.Time) (bool, error) {
	return r.model.ClaimTask(taskId, locker, now)
}

func (r *taskModelRepo) MarkSuccess(taskId int64) error {
	return r.model.MarkSuccess(taskId)
}

func (r *taskModelRepo) MarkRetry(taskId int64, attempts int, nextRetryAt time.Time, message string) error {
	return r.model.MarkRetry(taskId, attempts, nextRetryAt, message)
}

func (r *taskModelRepo) MarkDead(taskId int64, attempts int, message string) error {
	return r.model.MarkDead(taskId, attempts, message)
}

func (r *taskModelRepo) ReleaseClaim(taskId int64, nextRetryAt time.Time, message string) error {
	return r.model.ReleaseClaim(taskId, nextRetryAt, message)
}

type accountModelRepo struct {
	model *model.GeocodeAccountModel
}

func NewAccountRepo(m *model.GeocodeAccountModel) AccountRepository {
	return &accountModelRepo{model: m}
}

func (r *accountModelRepo) ListEnabled(provider string, now time.Time) ([]model.GeocodeAccount, error) {
	return r.model.ListEnabled(provider, now)
}

func (r *accountModelRepo) UpdateCooldown(accountId int64, coolDownUntil time.Time) error {
	return r.model.UpdateCooldown(accountId, coolDownUntil)
}

func (r *accountModelRepo) TouchSuccess(accountId int64) error {
	return r.model.TouchSuccess(accountId)
}

func (r *accountModelRepo) MarkDisabled(accountId int64) error {
	return r.model.MarkDisabled(accountId)
}

func (w *Worker) String() string {
	return fmt.Sprintf("Worker(provider=%s, lockerID=%s)", w.provider, w.lockerID)
}
