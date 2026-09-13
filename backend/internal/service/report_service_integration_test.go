package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/repository"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestReportServiceConcurrentCreate hammers a real database with concurrent
// report creations to prove the pending-unique constraint holds: at most one
// pending report survives per reporter and target, duplicates get a 409
// conflict, and the winning row is never overwritten. It runs only when
// REPORT_IT_DSN points at a disposable test database.
func TestReportServiceConcurrentCreate(t *testing.T) {
	dsn := os.Getenv("REPORT_IT_DSN")
	if dsn == "" {
		t.Skip("REPORT_IT_DSN not set; skipping concurrency integration test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS reports",
		"DROP TABLE IF EXISTS trade_orders",
		"DROP TABLE IF EXISTS products",
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("reset schema: %v", err)
		}
	}
	if err := db.AutoMigrate(&model.Product{}, &model.TradeOrder{}, &model.Report{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if _, err := repository.EnsureReportPendingUniqueIndex(db); err != nil {
		t.Fatalf("ensure pending unique index: %v", err)
	}

	ctx := context.Background()
	reportRepo := repository.NewReportRepository(db)
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewTradeOrderRepository(db)
	svc := NewReportService(reportRepo, productRepo, orderRepo, slog.Default())

	product := &model.Product{SellerID: 2, Title: "并发测试商品", Price: 10, Category: constants.ProductCategoryBooks, Status: constants.ProductStatusOnSale}
	if err := productRepo.Create(ctx, product); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	order := &model.TradeOrder{ProductID: product.ID, BuyerID: 1, SellerID: 2, Status: constants.TradeStatusCompleted}
	if err := orderRepo.Create(ctx, order); err != nil {
		t.Fatalf("seed completed order: %v", err)
	}

	// hammer fires n concurrent creates for the same reporter and target and
	// tallies successes, conflicts and unexpected errors.
	hammer := func(targetType string, targetID uint, n int) (success, conflict, other int) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.Create(context.Background(), 1, &dto.CreateReportRequest{
					TargetType: targetType, TargetID: targetID,
					Reason: constants.ReportReasonFake, Description: "并发举报测试",
				})
				mu.Lock()
				defer mu.Unlock()
				if err == nil {
					success++
					return
				}
				var appErr *util.AppError
				if errors.As(err, &appErr) && appErr.Status == 409 && appErr.Code == constants.CodeConflict {
					conflict++
					return
				}
				other++
			}()
		}
		wg.Wait()
		return success, conflict, other
	}

	countPending := func(targetType string, targetID uint) int64 {
		var n int64
		if err := db.Model(&model.Report{}).
			Where("reporter_id = ? AND target_type = ? AND target_id = ? AND status = ?", 1, targetType, targetID, constants.ReportStatusPending).
			Count(&n).Error; err != nil {
			t.Fatalf("count pending: %v", err)
		}
		return n
	}

	const concurrency = 16
	for _, target := range []struct {
		targetType string
		targetID   uint
	}{
		{constants.ReportTargetProduct, product.ID},
		{constants.ReportTargetTradeOrder, order.ID},
	} {
		t.Run(fmt.Sprintf("target_%s", target.targetType), func(t *testing.T) {
			success, conflict, other := hammer(target.targetType, target.targetID, concurrency)
			if other != 0 {
				t.Fatalf("unexpected errors during hammer: %d", other)
			}
			if success != 1 {
				t.Fatalf("expected exactly 1 success, got %d", success)
			}
			if conflict != concurrency-1 {
				t.Fatalf("expected %d conflicts, got %d", concurrency-1, conflict)
			}
			if got := countPending(target.targetType, target.targetID); got != 1 {
				t.Fatalf("expected exactly 1 pending row, got %d", got)
			}
		})
	}

	// A later duplicate with a different payload must be rejected and must not
	// overwrite the surviving record.
	var before model.Report
	if err := db.Where("reporter_id = ? AND target_type = ? AND target_id = ? AND status = ?",
		1, constants.ReportTargetProduct, product.ID, constants.ReportStatusPending).First(&before).Error; err != nil {
		t.Fatalf("load surviving report: %v", err)
	}
	_, err = svc.Create(ctx, 1, &dto.CreateReportRequest{
		TargetType: constants.ReportTargetProduct, TargetID: product.ID,
		Reason: constants.ReportReasonProhibited, Description: "试图覆盖原举报",
	})
	var appErr *util.AppError
	if err == nil || !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
		t.Fatalf("expected 409 conflict for duplicate, got %v", err)
	}
	var after model.Report
	if err := db.First(&after, before.ID).Error; err != nil {
		t.Fatalf("reload surviving report: %v", err)
	}
	if after.Reason != before.Reason || after.Description != before.Description || after.Status != before.Status {
		t.Fatalf("existing report overwritten: before=%+v after=%+v", before, after)
	}
}

// TestEnsureReportPendingUniqueIndexIdempotent verifies the migration can run
// repeatedly and dedups pre-existing duplicate pending rows exactly once.
func TestEnsureReportPendingUniqueIndexIdempotent(t *testing.T) {
	dsn := os.Getenv("REPORT_IT_DSN")
	if dsn == "" {
		t.Skip("REPORT_IT_DSN not set; skipping migration integration test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.Exec("DROP TABLE IF EXISTS reports").Error; err != nil {
		t.Fatalf("drop reports: %v", err)
	}
	if err := db.AutoMigrate(&model.Report{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	// plant duplicate pending rows bypassing the service layer
	for i := 0; i < 3; i++ {
		if err := db.Exec(`INSERT INTO reports (reporter_id, target_type, target_id, product_id, reason, description, status)
			VALUES (1, 'product', 7, 7, 'fake', '历史遗留重复', 'pending')`).Error; err != nil {
			t.Fatalf("plant duplicate: %v", err)
		}
	}
	deduped, err := repository.EnsureReportPendingUniqueIndex(db)
	if err != nil {
		t.Fatalf("first migration run: %v", err)
	}
	if deduped != 2 {
		t.Fatalf("expected 2 deduplicated rows, got %d", deduped)
	}
	var pending int64
	if err := db.Model(&model.Report{}).Where("status = ?", constants.ReportStatusPending).Count(&pending).Error; err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 1 {
		t.Fatalf("expected 1 pending row after dedup, got %d", pending)
	}
	// the surviving row must be the earliest one, untouched
	var survivor model.Report
	if err := db.Where("status = ?", constants.ReportStatusPending).First(&survivor).Error; err != nil {
		t.Fatalf("load survivor: %v", err)
	}
	if survivor.Description != "历史遗留重复" || survivor.Result != "" {
		t.Fatalf("surviving row was modified: %+v", survivor)
	}
	// second run is a no-op
	deduped, err = repository.EnsureReportPendingUniqueIndex(db)
	if err != nil {
		t.Fatalf("second migration run: %v", err)
	}
	if deduped != 0 {
		t.Fatalf("expected idempotent second run, deduped %d", deduped)
	}
}
