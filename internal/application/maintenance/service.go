package maintenance

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	domainprinters "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

var (
	ErrInvalidConfiguration = errors.New("invalid maintenance service configuration")
	ErrInvalidEvent         = errors.New("invalid maintenance event")
)
var maintenanceIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var maintenanceDecimalPattern = regexp.MustCompile(`^[0-9]{1,15}(\.[0-9]{1,3})?$`)

type Repository interface {
	Create(context.Context, string, string, domain.Values, time.Time) (domain.Event, error)
	List(context.Context, string) ([]domain.Event, error)
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
func (service *Service) Create(ctx context.Context, printerID, createdBy string, input domain.Values) (domain.Event, error) {
	printerID, pok := normalizeID(printerID)
	createdBy, uok := normalizeID(createdBy)
	if !pok {
		return domain.Event{}, domainprinters.ErrPrinterNotFound
	}
	if !uok {
		return domain.Event{}, ErrInvalidEvent
	}
	values, err := normalize(input)
	if err != nil {
		return domain.Event{}, err
	}
	event, err := service.repository.Create(ctx, printerID, createdBy, values, service.now().UTC())
	if err != nil {
		return domain.Event{}, fmt.Errorf("create maintenance event: %w", err)
	}
	return event, nil
}
func (service *Service) List(ctx context.Context, printerID string) ([]domain.Event, error) {
	printerID, ok := normalizeID(printerID)
	if !ok {
		return nil, domainprinters.ErrPrinterNotFound
	}
	events, err := service.repository.List(ctx, printerID)
	if err != nil {
		return nil, fmt.Errorf("list maintenance events: %w", err)
	}
	return events, nil
}
func normalize(values domain.Values) (domain.Values, error) {
	values.Description = strings.TrimSpace(values.Description)
	values.Notes = strings.TrimSpace(values.Notes)
	if values.PerformedAt.IsZero() || len(values.Description) == 0 || len(values.Description) > 10000 || len(values.Notes) > 10000 || values.DowntimeMinutes < 0 || (values.CostCents != nil && *values.CostCents < 0) || !validType(values.Type) {
		return domain.Values{}, ErrInvalidEvent
	}
	values.PerformedAt = values.PerformedAt.UTC()
	if values.PrinterHours != nil {
		value := normalizeDecimal(*values.PrinterHours)
		number, ok := new(big.Rat).SetString(value)
		if !maintenanceDecimalPattern.MatchString(value) || !ok || number.Sign() < 0 {
			return domain.Values{}, ErrInvalidEvent
		}
		values.PrinterHours = &value
	}
	return values, nil
}
func normalizeID(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, maintenanceIDPattern.MatchString(value)
}
func normalizeDecimal(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "" {
		return "0"
	}
	return value
}
func validType(value domain.Type) bool {
	switch value {
	case domain.TypeCleaning, domain.TypePreventive, domain.TypeCorrective, domain.TypeReplacement, domain.TypeUpgrade, domain.TypeInspection:
		return true
	}
	return false
}
