package labor

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
)

const laborJobID = "11111111-1111-4111-8111-111111111111"
const laborUserID = "22222222-2222-4222-8222-222222222222"
const laborRateID = "33333333-3333-4333-8333-333333333333"

func TestLaborSummaryTotalsMinutesByActivityWithoutCalculatingCost(t *testing.T) {
	repository := &laborRepositoryStub{entries: []domain.Entry{{ActivityType: domain.ActivitySetup, Minutes: 10, InternalHourlyRateCents: 6000}, {ActivityType: domain.ActivitySetup, Minutes: 5, InternalHourlyRateCents: 6000}, {ActivityType: domain.ActivityFinishing, Minutes: 20, InternalHourlyRateCents: 9000}}}
	service, _ := NewService(repository)
	summary, err := service.ListEntries(context.Background(), laborJobID)
	if err != nil || summary.TotalMinutes != 35 || summary.MinutesByActivity[domain.ActivitySetup] != 15 || summary.MinutesByActivity[domain.ActivityFinishing] != 20 {
		t.Fatalf("ListEntries()=%#v,%v", summary, err)
	}
}
func TestLaborEntryRequiresPositiveMinutesAndRate(t *testing.T) {
	service, _ := NewService(&laborRepositoryStub{})
	if _, err := service.CreateRate(context.Background(), domain.RateValues{Name: "Bad", ActivityType: domain.ActivitySetup, CostHourlyRateCents: -1}); !errors.Is(err, ErrInvalidRate) {
		t.Fatalf("CreateRate()=%v", err)
	}
	if _, err := service.CreateEntry(context.Background(), laborJobID, laborUserID, domain.EntryValues{LaborRateID: laborRateID, OccurredAt: time.Now()}); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("CreateEntry()=%v", err)
	}
}

type laborRepositoryStub struct{ entries []domain.Entry }

func (*laborRepositoryStub) CreateRate(_ context.Context, v domain.RateValues, _ time.Time) (domain.Rate, error) {
	return domain.Rate{Name: v.Name}, nil
}
func (*laborRepositoryStub) ListRates(context.Context) ([]domain.Rate, error) {
	return []domain.Rate{}, nil
}
func (*laborRepositoryStub) UpdateRate(context.Context, string, domain.RateValues, time.Time) (domain.Rate, error) {
	return domain.Rate{}, nil
}
func (*laborRepositoryStub) CreateEntry(_ context.Context, jobID, recordedBy string, v domain.EntryValues, _ time.Time) (domain.Entry, error) {
	return domain.Entry{JobID: jobID, LaborRateID: v.LaborRateID, Minutes: v.Minutes, RecordedBy: recordedBy}, nil
}
func (r *laborRepositoryStub) ListEntries(context.Context, string) ([]domain.Entry, error) {
	return r.entries, nil
}
