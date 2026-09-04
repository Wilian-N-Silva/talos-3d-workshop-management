package jobs

import (
	"context"
	"errors"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"testing"
	"time"
)

const jid = "11111111-1111-4111-8111-111111111111"

var actor = domain.Actor{UserID: "22222222-2222-4222-8222-222222222222", DeviceID: "33333333-3333-4333-8333-333333333333"}

func validValues() domain.Values {
	return domain.Values{Code: "JOB-1", CatalogItemID: "44444444-4444-4444-8444-444444444444", DesignVersionID: "55555555-5555-4555-8555-555555555555", PrinterID: "66666666-6666-4666-8666-666666666666", Purpose: domain.PurposeInternal, PlannedQuantity: 1}
}
func TestNonCommercialJobCreationDoesNotRequireOrder(t *testing.T) {
	repo := &repoStub{}
	service, _ := NewService(repo)
	job, err := service.Create(context.Background(), validValues(), actor)
	if err != nil || repo.values.Purpose != domain.PurposeInternal || job.OrderItemID != nil {
		t.Fatalf("Create()=%#v,%v values=%#v", job, err, repo.values)
	}
}
func TestTransitionRulesRequireQualityReviewForCompletion(t *testing.T) {
	repo := &repoStub{job: domain.Job{ID: jid, Status: domain.StatusDraft}}
	service, _ := NewService(repo)
	if _, err := service.Transition(context.Background(), jid, domain.TransitionValues{Status: domain.StatusCompleted}, actor); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("direct completion=%v", err)
	}
	if _, err := service.Transition(context.Background(), jid, domain.TransitionValues{Status: domain.StatusPrepared}, actor); err != nil || repo.event != domain.EventPrepared {
		t.Fatalf("prepare=%v event=%q", err, repo.event)
	}
}
func TestQualityReviewValidation(t *testing.T) {
	repo := &repoStub{job: domain.Job{ID: jid, Status: domain.StatusAwaitingReview}}
	service, _ := NewService(repo)
	if _, err := service.Review(context.Background(), jid, domain.ReviewValues{QualityStatus: domain.QualityPartial, GoodQuantity: 1, ScrapQuantity: 0}, actor); !errors.Is(err, ErrInvalidReview) {
		t.Fatalf("invalid partial=%v", err)
	}
	if _, err := service.Review(context.Background(), jid, domain.ReviewValues{QualityStatus: domain.QualityFailed, ScrapQuantity: 1}, actor); err != nil {
		t.Fatalf("failed review=%v", err)
	}
}

type repoStub struct {
	job    domain.Job
	values domain.Values
	event  domain.EventType
}

func (r *repoStub) Create(_ context.Context, v domain.Values, _ domain.Actor, _ time.Time) (domain.Job, error) {
	r.values = v
	return domain.Job{Purpose: v.Purpose}, nil
}
func (r *repoStub) FindByID(context.Context, string) (domain.Job, error) { return r.job, nil }
func (r *repoStub) List(context.Context) ([]domain.Job, error)           { return nil, nil }
func (r *repoStub) Update(context.Context, string, domain.Values, time.Time) (domain.Job, error) {
	return r.job, nil
}
func (r *repoStub) Transition(_ context.Context, _ string, _ domain.Status, _ domain.TransitionValues, e domain.EventType, _ domain.Actor, _ time.Time) (domain.Job, error) {
	r.event = e
	return r.job, nil
}
func (r *repoStub) Review(context.Context, string, domain.ReviewValues, domain.Actor, time.Time) (domain.Job, error) {
	return r.job, nil
}
func (r *repoStub) Delete(context.Context, string) error                       { return nil }
func (r *repoStub) ListEvents(context.Context, string) ([]domain.Event, error) { return nil, nil }
