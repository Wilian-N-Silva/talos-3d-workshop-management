package energy

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

var (
	ErrInvalidConfiguration = errors.New("invalid energy service configuration")
	ErrInvalidMeasurement   = errors.New("invalid energy measurement")
)

var energyDecimalPattern = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,6})?$`)
var energyIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Repository interface {
	Create(context.Context, string, string, domain.Values) (domain.Measurement, error)
	List(context.Context, string) ([]domain.Measurement, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Service{repository: repository}, nil
}

func (service *Service) Create(ctx context.Context, jobID, recordedBy string, input domain.Values) (domain.Measurement, error) {
	jobID = strings.ToLower(strings.TrimSpace(jobID))
	recordedBy = strings.ToLower(strings.TrimSpace(recordedBy))
	if !energyIDPattern.MatchString(jobID) {
		return domain.Measurement{}, domainjobs.ErrJobNotFound
	}
	if !energyIDPattern.MatchString(recordedBy) {
		return domain.Measurement{}, ErrInvalidMeasurement
	}
	values, err := normalize(input)
	if err != nil {
		return domain.Measurement{}, err
	}
	measurement, err := service.repository.Create(ctx, jobID, recordedBy, values)
	if err != nil {
		return domain.Measurement{}, fmt.Errorf("create energy measurement: %w", err)
	}
	return measurement, nil
}

func (service *Service) List(ctx context.Context, jobID string) ([]domain.Measurement, error) {
	jobID = strings.ToLower(strings.TrimSpace(jobID))
	if !energyIDPattern.MatchString(jobID) {
		return nil, domainjobs.ErrJobNotFound
	}
	measurements, err := service.repository.List(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("list energy measurements: %w", err)
	}
	return measurements, nil
}

func normalize(values domain.Values) (domain.Values, error) {
	values.Notes = strings.TrimSpace(values.Notes)
	if values.OccurredAt.IsZero() || len(values.Notes) > 10000 || values.EnergyRateCentsPerKWh < 0 || !validSource(values.Source) {
		return domain.Values{}, ErrInvalidMeasurement
	}
	values.OccurredAt = values.OccurredAt.UTC()
	for _, value := range []**string{&values.MeterStartKWh, &values.MeterEndKWh, &values.MeasuredKWh, &values.EstimatedAveragePowerW} {
		if *value == nil {
			continue
		}
		normalized := normalizeDecimal(**value)
		if !validDecimal(normalized) {
			return domain.Values{}, ErrInvalidMeasurement
		}
		*value = &normalized
	}
	switch values.Source {
	case domain.SourceManualMeter:
		if values.MeterStartKWh == nil || values.MeterEndKWh == nil || values.MeasuredKWh != nil || values.EstimatedAveragePowerW != nil {
			return domain.Values{}, ErrInvalidMeasurement
		}
		start, _ := new(big.Rat).SetString(*values.MeterStartKWh)
		end, _ := new(big.Rat).SetString(*values.MeterEndKWh)
		measured := new(big.Rat).Sub(end, start)
		if measured.Sign() < 0 {
			return domain.Values{}, ErrInvalidMeasurement
		}
		canonical := normalizeDecimal(measured.FloatString(6))
		values.MeasuredKWh = &canonical
	case domain.SourceSmartPlug, domain.SourceImported:
		if values.MeasuredKWh == nil || values.MeterStartKWh != nil || values.MeterEndKWh != nil || values.EstimatedAveragePowerW != nil {
			return domain.Values{}, ErrInvalidMeasurement
		}
	case domain.SourceEstimated:
		if values.EstimatedAveragePowerW == nil || values.MeterStartKWh != nil || values.MeterEndKWh != nil || values.MeasuredKWh != nil || decimalIsZero(*values.EstimatedAveragePowerW) {
			return domain.Values{}, ErrInvalidMeasurement
		}
	}
	return values, nil
}

func validSource(value domain.Source) bool {
	return value == domain.SourceManualMeter || value == domain.SourceSmartPlug || value == domain.SourceEstimated || value == domain.SourceImported
}
func validDecimal(value string) bool {
	number, ok := new(big.Rat).SetString(value)
	return energyDecimalPattern.MatchString(value) && ok && number.Sign() >= 0
}
func decimalIsZero(value string) bool {
	number, ok := new(big.Rat).SetString(value)
	return !ok || number.Sign() == 0
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
