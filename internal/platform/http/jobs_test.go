package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

const jobBody = `{"code":"JOB-1","catalog_item_id":"44444444-4444-4444-8444-444444444444","design_version_id":"55555555-5555-4555-8555-555555555555","printer_id":"66666666-6666-4666-8666-666666666666","purpose":"internal","planned_quantity":1,"hypothesis":"fixture","planned_seconds":3600,"labor_minutes":10}`

func TestJobRoutesAllowNonCommercialCreationAndCaptureActorDevice(t *testing.T) {
	service := &jobServiceHTTPStub{job: domain.Job{ID: "job-id", Purpose: domain.PurposeInternal, Status: domain.StatusDraft, QualityStatus: domain.QualityPending}}
	router := NewAPIV1Router()
	authentication := &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{User: domainauth.User{ID: "user-id", Status: domainauth.UserStatusActive, Role: domainauth.RoleOperator}, Session: domainauth.Session{DeviceID: "device-id"}}}
	RegisterJobs(router, authentication, service)
	created := inventoryRequest(router, http.MethodPost, JobsPath, jobBody)
	if created.Code != http.StatusCreated || service.values.Purpose != domain.PurposeInternal || service.actor.UserID != "user-id" || service.actor.DeviceID != "device-id" || strings.Contains(created.Body.String(), "revenue") {
		t.Fatalf("create=%d values=%#v actor=%#v body=%s", created.Code, service.values, service.actor, created.Body.String())
	}
	transition := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/transitions", `{"status":"prepared","result_notes":""}`)
	if transition.Code != http.StatusOK || service.transition.Status != domain.StatusPrepared {
		t.Fatalf("transition=%d values=%#v body=%s", transition.Code, service.transition, transition.Body.String())
	}
	review := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/review", `{"quality_status":"partial","good_quantity":1,"scrap_quantity":1,"result_notes":"one rejected"}`)
	if review.Code != http.StatusOK || service.review.QualityStatus != domain.QualityPartial {
		t.Fatalf("review=%d values=%#v body=%s", review.Code, service.review, review.Body.String())
	}
}
func TestJobCreatePermissionIsEnforced(t *testing.T) {
	router := NewAPIV1Router()
	RegisterJobs(router, authorizedCatalogUser(domainauth.RoleViewer), &jobServiceHTTPStub{})
	response := inventoryRequest(router, http.MethodPost, JobsPath, jobBody)
	assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
}

type jobServiceHTTPStub struct {
	job        domain.Job
	values     domain.Values
	actor      domain.Actor
	transition domain.TransitionValues
	review     domain.ReviewValues
}

func (s *jobServiceHTTPStub) Create(_ context.Context, v domain.Values, a domain.Actor) (domain.Job, error) {
	s.values, s.actor = v, a
	return s.job, nil
}
func (s *jobServiceHTTPStub) Get(context.Context, string) (domain.Job, error) { return s.job, nil }
func (s *jobServiceHTTPStub) List(context.Context) ([]domain.Job, error) {
	return []domain.Job{s.job}, nil
}
func (s *jobServiceHTTPStub) Update(_ context.Context, _ string, v domain.Values) (domain.Job, error) {
	s.values = v
	return s.job, nil
}
func (s *jobServiceHTTPStub) Transition(_ context.Context, _ string, v domain.TransitionValues, a domain.Actor) (domain.Job, error) {
	s.transition, s.actor = v, a
	return s.job, nil
}
func (s *jobServiceHTTPStub) Review(_ context.Context, _ string, v domain.ReviewValues, a domain.Actor) (domain.Job, error) {
	s.review, s.actor = v, a
	return s.job, nil
}
func (s *jobServiceHTTPStub) Delete(context.Context, string) error { return nil }
func (s *jobServiceHTTPStub) ListEvents(context.Context, string) ([]domain.Event, error) {
	return []domain.Event{}, nil
}
