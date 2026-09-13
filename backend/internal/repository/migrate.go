package repository

import (
	"fmt"

	"gorm.io/gorm"
)

// Schema objects backing the "one pending report per reporter and target" rule.
const (
	reportPendingKeyColumn   = "pending_key"
	reportPendingUniqueIndex = "uniq_reports_pending_key"
)

// EnsureReportPendingUniqueIndex hardens the reports table so that concurrent
// inserts cannot create two pending reports for the same reporter and target.
// MySQL has no partial unique indexes, so a generated column yields the unique
// key only for pending rows (handled rows collapse to NULL and never conflict).
// The migration is idempotent and returns how many duplicate pending rows were
// auto-rejected before the unique index was created.
func EnsureReportPendingUniqueIndex(db *gorm.DB) (int64, error) {
	exists, err := columnExists(db, "reports", reportPendingKeyColumn)
	if err != nil {
		return 0, fmt.Errorf("check reports.%s column: %w", reportPendingKeyColumn, err)
	}
	if !exists {
		if err := db.Exec(`ALTER TABLE reports ADD COLUMN pending_key VARCHAR(80)
			GENERATED ALWAYS AS (IF(status = 'pending', CONCAT(target_type, ':', target_id, ':', reporter_id), NULL)) VIRTUAL`).Error; err != nil {
			return 0, fmt.Errorf("add reports.%s column: %w", reportPendingKeyColumn, err)
		}
	}
	// Keep the earliest pending report per reporter and target; auto-reject the
	// rest so the unique index can be created. Existing rows are never updated
	// beyond this one-time dedup.
	dedup := db.Exec(`UPDATE reports r1 JOIN reports r2
			ON r1.reporter_id = r2.reporter_id
			AND r1.target_type = r2.target_type
			AND r1.target_id = r2.target_id
			AND r1.status = 'pending'
			AND r2.status = 'pending'
			AND r1.id > r2.id
		SET r1.status = 'rejected',
			r1.result = '系统自动去重：仅保留最早一条待处理举报',
			r1.handled_at = NOW(3)`)
	if dedup.Error != nil {
		return 0, fmt.Errorf("dedup pending reports: %w", dedup.Error)
	}
	exists, err = indexExists(db, "reports", reportPendingUniqueIndex)
	if err != nil {
		return 0, fmt.Errorf("check reports.%s index: %w", reportPendingUniqueIndex, err)
	}
	if !exists {
		if err := db.Exec(`ALTER TABLE reports ADD UNIQUE INDEX uniq_reports_pending_key (pending_key)`).Error; err != nil {
			return 0, fmt.Errorf("add reports.%s index: %w", reportPendingUniqueIndex, err)
		}
	}
	return dedup.RowsAffected, nil
}

// columnExists reports whether the table has the given column.
func columnExists(db *gorm.DB, table, column string) (bool, error) {
	var n int64
	err := db.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column).Scan(&n).Error
	return n > 0, err
}

// indexExists reports whether the table has the given index.
func indexExists(db *gorm.DB, table, index string) (bool, error) {
	var n int64
	err := db.Raw(`SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`, table, index).Scan(&n).Error
	return n > 0, err
}
