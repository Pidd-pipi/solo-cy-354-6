package repository

import (
	"context"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
)

// ReportRepository persists report rows.
type ReportRepository struct {
	db *gorm.DB
}

// NewReportRepository builds a ReportRepository.
func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Transaction runs fn inside a database transaction for cross-repository writes.
func (r *ReportRepository) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return Transaction(ctx, r.db, fn)
}

// Create inserts a new report.
func (r *ReportRepository) Create(ctx context.Context, rp *model.Report) error {
	return db(ctx, r.db).Create(rp).Error
}

// FindByID returns a report by id.
func (r *ReportRepository) FindByID(ctx context.Context, id uint) (*model.Report, error) {
	var rp model.Report
	err := db(ctx, r.db).First(&rp, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// FindPendingByReporterAndTarget returns the pending report of a reporter for
// one target, enforcing the "one pending report per reporter and target" rule.
func (r *ReportRepository) FindPendingByReporterAndTarget(ctx context.Context, reporterID uint, targetType string, targetID uint) (*model.Report, error) {
	var rp model.Report
	err := db(ctx, r.db).
		Where("reporter_id = ? AND target_type = ? AND target_id = ? AND status = ?", reporterID, targetType, targetID, constants.ReportStatusPending).
		First(&rp).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &rp, nil
}

// List filters reports by status with pagination.
func (r *ReportRepository) List(ctx context.Context, status string, page, pageSize int) ([]model.Report, int64, error) {
	q := db(ctx, r.db).Model(&model.Report{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Report
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkHandled sets the handling outcome of a pending report. It returns
// util.ErrConflict when the report is no longer pending.
func (r *ReportRepository) MarkHandled(ctx context.Context, id uint, status, result string, handledBy uint, handledAt time.Time) error {
	res := db(ctx, r.db).Model(&model.Report{}).Where("id = ? AND status = ?", id, constants.ReportStatusPending).
		Updates(map[string]interface{}{
			"status":     status,
			"result":     result,
			"handled_by": handledBy,
			"handled_at": handledAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}
