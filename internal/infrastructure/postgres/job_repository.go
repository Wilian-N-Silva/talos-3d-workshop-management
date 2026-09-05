package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

const jobColumns = `id,code,catalog_item_id,design_version_id,printer_id,order_item_id,purpose,status,planned_quantity,good_quantity,scrap_quantity,hypothesis,result_notes,quality_status,planned_seconds,actual_seconds,labor_minutes,created_by,created_at,updated_at,started_at,completed_at`

type JobRepository struct{ database *sql.DB }

func NewJobRepository(database *sql.DB) *JobRepository { return &JobRepository{database: database} }

func (r *JobRepository) Create(ctx context.Context, v domain.Values, actor domain.Actor, now time.Time) (domain.Job, error) {
	tx, err := r.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Job{}, fmt.Errorf("begin job create: %w", err)
	}
	defer tx.Rollback()
	var valid bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM design_versions d JOIN catalog_parts p ON p.id=d.catalog_part_id WHERE d.id=$1 AND p.catalog_item_id=$2) AND EXISTS(SELECT 1 FROM printers WHERE id=$3)`, v.DesignVersionID, v.CatalogItemID, v.PrinterID).Scan(&valid); err != nil {
		return domain.Job{}, fmt.Errorf("validate job references: %w", err)
	}
	if !valid {
		return domain.Job{}, domain.ErrJobReference
	}
	job, err := scanJob(tx.QueryRowContext(ctx, `INSERT INTO print_jobs (code,catalog_item_id,design_version_id,printer_id,purpose,planned_quantity,hypothesis,planned_seconds,labor_minutes,created_by,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11) RETURNING `+jobColumns, v.Code, v.CatalogItemID, v.DesignVersionID, v.PrinterID, v.Purpose, v.PlannedQuantity, v.Hypothesis, v.PlannedSeconds, v.LaborMinutes, actor.UserID, now.UTC()))
	if uniqueError(err, "print_jobs_code_unique") {
		return domain.Job{}, domain.ErrJobCodeConflict
	}
	if fkError(err, "print_jobs_created_by_fk") {
		return domain.Job{}, domain.ErrJobReference
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("insert print job: %w", err)
	}
	if err := insertJobEvent(ctx, tx, job.ID, domain.EventCreated, actor, now, json.RawMessage(`{}`)); err != nil {
		return domain.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Job{}, fmt.Errorf("commit job create: %w", err)
	}
	return job, nil
}

func (r *JobRepository) FindByID(ctx context.Context, id string) (domain.Job, error) {
	job, err := scanJob(r.database.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM print_jobs WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, domain.ErrJobNotFound
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("find print job: %w", err)
	}
	return job, nil
}
func (r *JobRepository) List(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT `+jobColumns+` FROM print_jobs ORDER BY created_at DESC,id DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query print jobs: %w", err)
	}
	defer rows.Close()
	result := []domain.Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan print job: %w", err)
		}
		result = append(result, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate print jobs: %w", err)
	}
	return result, nil
}
func (r *JobRepository) Update(ctx context.Context, id string, v domain.Values, now time.Time) (domain.Job, error) {
	var valid bool
	if err := r.database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM design_versions d JOIN catalog_parts p ON p.id=d.catalog_part_id WHERE d.id=$1 AND p.catalog_item_id=$2) AND EXISTS(SELECT 1 FROM printers WHERE id=$3)`, v.DesignVersionID, v.CatalogItemID, v.PrinterID).Scan(&valid); err != nil {
		return domain.Job{}, fmt.Errorf("validate job references: %w", err)
	}
	if !valid {
		return domain.Job{}, domain.ErrJobReference
	}
	job, err := scanJob(r.database.QueryRowContext(ctx, `UPDATE print_jobs SET code=$2,catalog_item_id=$3,design_version_id=$4,printer_id=$5,purpose=$6,planned_quantity=$7,hypothesis=$8,planned_seconds=$9,labor_minutes=$10,updated_at=GREATEST(updated_at,$11) WHERE id=$1 AND status IN ('draft','prepared') RETURNING `+jobColumns, id, v.Code, v.CatalogItemID, v.DesignVersionID, v.PrinterID, v.Purpose, v.PlannedQuantity, v.Hypothesis, v.PlannedSeconds, v.LaborMinutes, now.UTC()))
	if uniqueError(err, "print_jobs_code_unique") {
		return domain.Job{}, domain.ErrJobCodeConflict
	}
	if fkError(err, "print_jobs_printer_fk") {
		return domain.Job{}, domain.ErrJobReference
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, r.missingOrState(ctx, id, domain.ErrJobNotEditable)
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("update print job: %w", err)
	}
	return job, nil
}
func (r *JobRepository) Transition(ctx context.Context, id string, expected domain.Status, v domain.TransitionValues, event domain.EventType, actor domain.Actor, now time.Time) (domain.Job, error) {
	tx, err := r.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Job{}, fmt.Errorf("begin job transition: %w", err)
	}
	defer tx.Rollback()
	job, err := scanJob(tx.QueryRowContext(ctx, `UPDATE print_jobs SET status=$3::text,actual_seconds=COALESCE($4,actual_seconds),result_notes=$5,started_at=CASE WHEN $3::text='printing' THEN COALESCE(started_at,$6) ELSE started_at END,completed_at=CASE WHEN $3::text IN ('failed','cancelled') THEN $6 ELSE completed_at END,updated_at=GREATEST(updated_at,$6) WHERE id=$1 AND status=$2 RETURNING `+jobColumns, id, expected, v.Status, v.ActualSeconds, v.ResultNotes, now.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, r.missingOrStateTx(ctx, tx, id, domain.ErrJobStateConflict)
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("update job transition: %w", err)
	}
	metadata, _ := json.Marshal(map[string]any{"from": expected, "to": v.Status})
	if err := insertJobEvent(ctx, tx, id, event, actor, now, metadata); err != nil {
		return domain.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Job{}, fmt.Errorf("commit job transition: %w", err)
	}
	return job, nil
}
func (r *JobRepository) Review(ctx context.Context, id string, v domain.ReviewValues, actor domain.Actor, now time.Time) (domain.Job, error) {
	tx, err := r.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Job{}, fmt.Errorf("begin job review: %w", err)
	}
	defer tx.Rollback()
	status := domain.StatusCompleted
	if v.QualityStatus == domain.QualityFailed {
		status = domain.StatusFailed
	}
	job, err := scanJob(tx.QueryRowContext(ctx, `UPDATE print_jobs SET status=$2,quality_status=$3,good_quantity=$4,scrap_quantity=$5,result_notes=$6,completed_at=$7,updated_at=GREATEST(updated_at,$7) WHERE id=$1 AND status='awaiting_review' RETURNING `+jobColumns, id, status, v.QualityStatus, v.GoodQuantity, v.ScrapQuantity, v.ResultNotes, now.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, r.missingOrStateTx(ctx, tx, id, domain.ErrJobStateConflict)
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("update job review: %w", err)
	}
	metadata, _ := json.Marshal(map[string]any{"quality_status": v.QualityStatus, "good_quantity": v.GoodQuantity, "scrap_quantity": v.ScrapQuantity})
	if err := insertJobEvent(ctx, tx, id, domain.EventReviewed, actor, now, metadata); err != nil {
		return domain.Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Job{}, fmt.Errorf("commit job review: %w", err)
	}
	return job, nil
}
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	result, err := r.database.ExecContext(ctx, "DELETE FROM print_jobs WHERE id=$1 AND status='draft'", id)
	if err != nil {
		return fmt.Errorf("delete print job: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted job count: %w", err)
	}
	if rows == 0 {
		return r.missingOrState(ctx, id, domain.ErrJobNotDeletable)
	}
	return nil
}
func (r *JobRepository) ListEvents(ctx context.Context, id string) ([]domain.Event, error) {
	var exists bool
	if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", id).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return nil, domain.ErrJobNotFound
	}
	rows, err := r.database.QueryContext(ctx, `SELECT id,job_id,event_type,occurred_at,actor_user_id,source_device_id,metadata FROM job_events WHERE job_id=$1 ORDER BY occurred_at,id LIMIT 200`, id)
	if err != nil {
		return nil, fmt.Errorf("query job events: %w", err)
	}
	defer rows.Close()
	result := []domain.Event{}
	for rows.Next() {
		var event domain.Event
		if err := rows.Scan(&event.ID, &event.JobID, &event.Type, &event.OccurredAt, &event.ActorUserID, &event.SourceDeviceID, &event.Metadata); err != nil {
			return nil, fmt.Errorf("scan job event: %w", err)
		}
		event.OccurredAt = event.OccurredAt.UTC()
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job events: %w", err)
	}
	return result, nil
}

func insertJobEvent(ctx context.Context, tx *sql.Tx, id string, event domain.EventType, actor domain.Actor, now time.Time, metadata json.RawMessage) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO job_events(job_id,event_type,occurred_at,actor_user_id,source_device_id,metadata) VALUES($1,$2,$3,$4,$5,$6)`, id, event, now.UTC(), actor.UserID, actor.DeviceID, metadata); err != nil {
		return fmt.Errorf("insert job event: %w", err)
	}
	return nil
}
func (r *JobRepository) missingOrState(ctx context.Context, id string, state error) error {
	var exists bool
	if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", id).Scan(&exists); err != nil {
		return fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return domain.ErrJobNotFound
	}
	return state
}
func (r *JobRepository) missingOrStateTx(ctx context.Context, tx *sql.Tx, id string, state error) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", id).Scan(&exists); err != nil {
		return fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return domain.ErrJobNotFound
	}
	return state
}

func scanJob(row rowScanner) (domain.Job, error) {
	var job domain.Job
	var order sql.NullString
	var actual sql.NullInt64
	var started, completed sql.NullTime
	if err := row.Scan(&job.ID, &job.Code, &job.CatalogItemID, &job.DesignVersionID, &job.PrinterID, &order, &job.Purpose, &job.Status, &job.PlannedQuantity, &job.GoodQuantity, &job.ScrapQuantity, &job.Hypothesis, &job.ResultNotes, &job.QualityStatus, &job.PlannedSeconds, &actual, &job.LaborMinutes, &job.CreatedBy, &job.CreatedAt, &job.UpdatedAt, &started, &completed); err != nil {
		return domain.Job{}, err
	}
	if order.Valid {
		job.OrderItemID = &order.String
	}
	if actual.Valid {
		job.ActualSeconds = &actual.Int64
	}
	if started.Valid {
		value := started.Time.UTC()
		job.StartedAt = &value
	}
	if completed.Valid {
		value := completed.Time.UTC()
		job.CompletedAt = &value
	}
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	return job, nil
}
