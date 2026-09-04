package printers

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

var (
	ErrInvalidPrinter       = errors.New("invalid printer")
	ErrInvalidConfiguration = errors.New("invalid printer service configuration")
	printerIDPattern        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	nozzlePattern           = regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]{1,3})?$`)
	usefulLifePattern       = regexp.MustCompile(`^[0-9]{1,10}(\.[0-9]{1,2})?$`)
)

type Repository interface {
	Create(context.Context, domain.Values, time.Time) (domain.Printer, error)
	FindByID(context.Context, string) (domain.Printer, error)
	List(context.Context) ([]domain.Printer, error)
	Update(context.Context, string, domain.Values, time.Time) (domain.Printer, error)
	Delete(context.Context, string) error
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

func (service *Service) Create(ctx context.Context, input domain.Values) (domain.Printer, error) {
	values, err := normalize(input)
	if err != nil {
		return domain.Printer{}, err
	}
	printer, err := service.repository.Create(ctx, values, service.now().UTC())
	if err != nil {
		return domain.Printer{}, fmt.Errorf("create printer: %w", err)
	}
	return printer, nil
}

func (service *Service) Get(ctx context.Context, id string) (domain.Printer, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !printerIDPattern.MatchString(id) {
		return domain.Printer{}, domain.ErrPrinterNotFound
	}
	printer, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Printer{}, fmt.Errorf("get printer: %w", err)
	}
	return printer, nil
}

func (service *Service) List(ctx context.Context) ([]domain.Printer, error) {
	printers, err := service.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list printers: %w", err)
	}
	return printers, nil
}

func (service *Service) Update(ctx context.Context, id string, input domain.Values) (domain.Printer, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !printerIDPattern.MatchString(id) {
		return domain.Printer{}, domain.ErrPrinterNotFound
	}
	values, err := normalize(input)
	if err != nil {
		return domain.Printer{}, err
	}
	printer, err := service.repository.Update(ctx, id, values, service.now().UTC())
	if err != nil {
		return domain.Printer{}, fmt.Errorf("update printer: %w", err)
	}
	return printer, nil
}

func (service *Service) Delete(ctx context.Context, id string) error {
	id = strings.ToLower(strings.TrimSpace(id))
	if !printerIDPattern.MatchString(id) {
		return domain.ErrPrinterNotFound
	}
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete printer: %w", err)
	}
	return nil
}

func normalize(input domain.Values) (domain.Values, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Manufacturer = strings.TrimSpace(input.Manufacturer)
	input.Model = strings.TrimSpace(input.Model)
	input.NozzleDiameter = normalizeDecimal(input.NozzleDiameter)
	input.Location = strings.TrimSpace(input.Location)
	input.UsefulLifeHours = normalizeDecimal(input.UsefulLifeHours)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Status == "" {
		input.Status = domain.StatusActive
	}
	nozzle, nozzleOK := new(big.Rat).SetString(input.NozzleDiameter)
	usefulLife, usefulLifeOK := new(big.Rat).SetString(input.UsefulLifeHours)
	if input.Name == "" || len(input.Name) > 200 || input.Manufacturer == "" || len(input.Manufacturer) > 200 || input.Model == "" || len(input.Model) > 200 || len(input.Location) > 500 || len(input.Notes) > 10000 || !nozzlePattern.MatchString(input.NozzleDiameter) || !nozzleOK || nozzle.Sign() <= 0 || !usefulLifePattern.MatchString(input.UsefulLifeHours) || !usefulLifeOK || usefulLife.Sign() <= 0 || input.AcquisitionCostCents < 0 || input.ResidualValueCents < 0 || input.ResidualValueCents > input.AcquisitionCostCents || input.MaintenanceReservePerHourCents < 0 || !validStatus(input.Status) {
		return domain.Values{}, ErrInvalidPrinter
	}
	return input, nil
}

func validStatus(status domain.Status) bool {
	return status == domain.StatusActive || status == domain.StatusMaintenance || status == domain.StatusRetired
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
