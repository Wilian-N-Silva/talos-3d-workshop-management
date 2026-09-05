package jobs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

var (
	ErrInvalidJob           = errors.New("invalid print job")
	ErrInvalidTransition    = errors.New("invalid print job transition")
	ErrInvalidReview        = errors.New("invalid print job review")
	ErrInvalidConfiguration = errors.New("invalid print job service configuration")
	jobIDPattern            = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type Repository interface {
	Create(context.Context, domain.Values, domain.Actor, time.Time) (domain.Job, error)
	FindByID(context.Context, string) (domain.Job, error)
	List(context.Context) ([]domain.Job, error)
	Update(context.Context, string, domain.Values, time.Time) (domain.Job, error)
	Transition(context.Context, string, domain.Status, domain.TransitionValues, domain.EventType, domain.Actor, time.Time) (domain.Job, error)
	Review(context.Context, string, domain.ReviewValues, domain.Actor, time.Time) (domain.Job, error)
	Delete(context.Context, string) error
	ListEvents(context.Context, string) ([]domain.Event, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Service{repository: repository, now: time.Now}, nil
}

func (s *Service) Create(ctx context.Context, input domain.Values, actor domain.Actor) (domain.Job, error) {
	values, err := normalizeValues(input)
	if err != nil || !validActor(actor) {
		return domain.Job{}, ErrInvalidJob
	}
	job, err := s.repository.Create(ctx, values, normalizeActor(actor), s.now().UTC())
	if err != nil {
		return domain.Job{}, fmt.Errorf("create print job: %w", err)
	}
	return job, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Job, error) {
	id, ok := normalizeID(id)
	if !ok {
		return domain.Job{}, domain.ErrJobNotFound
	}
	job, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Job{}, fmt.Errorf("get print job: %w", err)
	}
	return job, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Job, error) {
	jobs, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list print jobs: %w", err)
	}
	return jobs, nil
}

func (s *Service) Update(ctx context.Context, id string, input domain.Values) (domain.Job, error) {
	id, ok := normalizeID(id)
	if !ok {
		return domain.Job{}, domain.ErrJobNotFound
	}
	values, err := normalizeValues(input)
	if err != nil {
		return domain.Job{}, err
	}
	job, err := s.repository.Update(ctx, id, values, s.now().UTC())
	if err != nil {
		return domain.Job{}, fmt.Errorf("update print job: %w", err)
	}
	return job, nil
}

func (s *Service) Transition(ctx context.Context, id string, input domain.TransitionValues, actor domain.Actor) (domain.Job, error) {
	id, ok := normalizeID(id)
	if !ok {
		return domain.Job{}, domain.ErrJobNotFound
	}
	if !validActor(actor) || len(strings.TrimSpace(input.ResultNotes)) > 10000 || (input.ActualSeconds != nil && *input.ActualSeconds < 0) {
		return domain.Job{}, ErrInvalidTransition
	}
	current, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Job{}, fmt.Errorf("get print job for transition: %w", err)
	}
	event, ok := transitionEvent(current.Status, input.Status)
	if !ok {
		return domain.Job{}, ErrInvalidTransition
	}
	if input.ActualSeconds != nil && input.Status != domain.StatusAwaitingReview && input.Status != domain.StatusFailed {
		return domain.Job{}, ErrInvalidTransition
	}
	input.ResultNotes = strings.TrimSpace(input.ResultNotes)
	job, err := s.repository.Transition(ctx, id, current.Status, input, event, normalizeActor(actor), s.now().UTC())
	if err != nil {
		return domain.Job{}, fmt.Errorf("transition print job: %w", err)
	}
	return job, nil
}

func (s *Service) Review(ctx context.Context, id string, input domain.ReviewValues, actor domain.Actor) (domain.Job, error) {
	id, ok := normalizeID(id)
	if !ok {
		return domain.Job{}, domain.ErrJobNotFound
	}
	input.ResultNotes = strings.TrimSpace(input.ResultNotes)
	if !validActor(actor) || input.GoodQuantity < 0 || input.ScrapQuantity < 0 || len(input.ResultNotes) > 10000 || !validReview(input) {
		return domain.Job{}, ErrInvalidReview
	}
	job, err := s.repository.Review(ctx, id, input, normalizeActor(actor), s.now().UTC())
	if err != nil {
		return domain.Job{}, fmt.Errorf("review print job: %w", err)
	}
	return job, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	id, ok := normalizeID(id)
	if !ok {
		return domain.ErrJobNotFound
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete print job: %w", err)
	}
	return nil
}

func (s *Service) ListEvents(ctx context.Context, id string) ([]domain.Event, error) {
	id, ok := normalizeID(id)
	if !ok {
		return nil, domain.ErrJobNotFound
	}
	events, err := s.repository.ListEvents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list print job events: %w", err)
	}
	return events, nil
}

func normalizeValues(v domain.Values) (domain.Values, error) {
	v.Code = strings.TrimSpace(v.Code)
	v.Hypothesis = strings.TrimSpace(v.Hypothesis)
	var ok bool
	if v.CatalogItemID, ok = normalizedRequiredID(v.CatalogItemID); !ok {
		return domain.Values{}, ErrInvalidJob
	}
	if v.DesignVersionID, ok = normalizedRequiredID(v.DesignVersionID); !ok {
		return domain.Values{}, ErrInvalidJob
	}
	if v.PrinterID, ok = normalizedRequiredID(v.PrinterID); !ok {
		return domain.Values{}, ErrInvalidJob
	}
	if v.Code == "" || len(v.Code) > 100 || len(v.Hypothesis) > 10000 || v.PlannedQuantity <= 0 || v.PlannedSeconds < 0 || v.LaborMinutes < 0 || !validPurpose(v.Purpose) {
		return domain.Values{}, ErrInvalidJob
	}
	return v, nil
}

func validPurpose(v domain.Purpose) bool {
	return v == domain.PurposeTest || v == domain.PurposePrototype || v == domain.PurposeProduction || v == domain.PurposeMaintenance || v == domain.PurposeInternal || v == domain.PurposePersonal
}
func validActor(v domain.Actor) bool {
	_, a := normalizeID(v.UserID)
	_, b := normalizeID(v.DeviceID)
	return a && b
}
func normalizeActor(v domain.Actor) domain.Actor {
	v.UserID, _ = normalizeID(v.UserID)
	v.DeviceID, _ = normalizeID(v.DeviceID)
	return v
}
func normalizeID(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	return v, jobIDPattern.MatchString(v)
}
func normalizedRequiredID(v string) (string, bool) { return normalizeID(v) }

func transitionEvent(from, to domain.Status) (domain.EventType, bool) {
	switch {
	case from == domain.StatusDraft && to == domain.StatusPrepared:
		return domain.EventPrepared, true
	case from == domain.StatusPrepared && to == domain.StatusPrinting:
		return domain.EventPrintingStartedManual, true
	case from == domain.StatusPrinting && to == domain.StatusAwaitingReview:
		return domain.EventFinishedManual, true
	case (from == domain.StatusDraft || from == domain.StatusPrepared || from == domain.StatusPrinting) && to == domain.StatusCancelled:
		return domain.EventCancelled, true
	case from == domain.StatusPrinting && to == domain.StatusFailed:
		return domain.EventFailed, true
	default:
		return "", false
	}
}

func validReview(v domain.ReviewValues) bool {
	switch v.QualityStatus {
	case domain.QualityApproved:
		return v.GoodQuantity > 0
	case domain.QualityPartial:
		return v.GoodQuantity > 0 && v.ScrapQuantity > 0
	case domain.QualityFailed:
		return v.GoodQuantity == 0 && v.ScrapQuantity > 0
	default:
		return false
	}
}
