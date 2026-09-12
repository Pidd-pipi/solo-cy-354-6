package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

type fakeReportRepo struct {
	reports map[uint]*model.Report
	nextID  uint
}

func newFakeReportRepo() *fakeReportRepo {
	return &fakeReportRepo{reports: map[uint]*model.Report{}, nextID: 1}
}

func (f *fakeReportRepo) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func (f *fakeReportRepo) Create(_ context.Context, rp *model.Report) error {
	rp.ID = f.nextID
	f.nextID++
	f.reports[rp.ID] = rp
	return nil
}

func (f *fakeReportRepo) FindByID(_ context.Context, id uint) (*model.Report, error) {
	if rp, ok := f.reports[id]; ok {
		cp := *rp
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportRepo) FindPendingByReporterAndTarget(_ context.Context, reporterID uint, targetType string, targetID uint) (*model.Report, error) {
	for _, rp := range f.reports {
		if rp.ReporterID == reporterID && rp.TargetType == targetType && rp.TargetID == targetID && rp.Status == constants.ReportStatusPending {
			cp := *rp
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportRepo) List(_ context.Context, status string, _, _ int) ([]model.Report, int64, error) {
	var out []model.Report
	for _, rp := range f.reports {
		if status != "" && rp.Status != status {
			continue
		}
		out = append(out, *rp)
	}
	return out, int64(len(out)), nil
}

func (f *fakeReportRepo) MarkHandled(_ context.Context, id uint, status, result string, handledBy uint, handledAt time.Time) error {
	rp, ok := f.reports[id]
	if !ok || rp.Status != constants.ReportStatusPending {
		return util.ErrConflict
	}
	rp.Status = status
	rp.Result = result
	rp.HandledBy = handledBy
	rp.HandledAt = &handledAt
	return nil
}

type fakeReportProductRepo struct {
	products map[uint]*model.Product
}

func (f *fakeReportProductRepo) FindByID(_ context.Context, id uint) (*model.Product, error) {
	if p, ok := f.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeReportProductRepo) UpdateStatus(_ context.Context, id uint, status string) error {
	if p, ok := f.products[id]; ok {
		p.Status = status
		return nil
	}
	return util.ErrNotFound
}

type fakeReportOrderRepo struct {
	orders map[uint]*model.TradeOrder
}

func (f *fakeReportOrderRepo) FindByID(_ context.Context, id uint) (*model.TradeOrder, error) {
	if o, ok := f.orders[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func newReportFixture() (*ReportService, *fakeReportRepo, *fakeReportProductRepo) {
	reports := newFakeReportRepo()
	products := &fakeReportProductRepo{products: map[uint]*model.Product{
		1: {ID: 1, SellerID: 2, Title: "高数教材", Status: constants.ProductStatusOnSale},
		2: {ID: 2, SellerID: 3, Title: "台灯", Status: constants.ProductStatusOnSale},
	}}
	orders := &fakeReportOrderRepo{orders: map[uint]*model.TradeOrder{
		10: {ID: 10, ProductID: 2, BuyerID: 1, SellerID: 3, Status: constants.TradeStatusCompleted},
		11: {ID: 11, ProductID: 2, BuyerID: 1, SellerID: 3, Status: constants.TradeStatusPending},
	}}
	svc := NewReportService(reports, products, orders, slog.Default())
	return svc, reports, products
}

func TestReportServiceCreate(t *testing.T) {
	tests := []struct {
		name       string
		reporterID uint
		req        *dto.CreateReportRequest
		wantErr    bool
	}{
		{name: "report product", reporterID: 1, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 1, Reason: constants.ReportReasonFake, Description: "描述与实物不符"}, wantErr: false},
		{name: "report completed order as buyer", reporterID: 1, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetTradeOrder, TargetID: 10, Reason: constants.ReportReasonFraud, Description: "卖家收款后失联"}, wantErr: false},
		{name: "report completed order as seller", reporterID: 3, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetTradeOrder, TargetID: 10, Reason: constants.ReportReasonAbuse, Description: "买家恶意辱骂"}, wantErr: false},
		{name: "report pending order rejected", reporterID: 1, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetTradeOrder, TargetID: 11, Reason: constants.ReportReasonOther, Description: "订单未完成"}, wantErr: true},
		{name: "report order as outsider rejected", reporterID: 99, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetTradeOrder, TargetID: 10, Reason: constants.ReportReasonOther, Description: "非交易双方"}, wantErr: true},
		{name: "report missing product rejected", reporterID: 1, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 999, Reason: constants.ReportReasonOther, Description: "商品不存在"}, wantErr: true},
		{name: "invalid reason rejected", reporterID: 1, req: &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 1, Reason: "nonsense", Description: "理由不合法"}, wantErr: true},
		{name: "invalid target type rejected", reporterID: 1, req: &dto.CreateReportRequest{TargetType: "user", TargetID: 1, Reason: constants.ReportReasonOther, Description: "对象不合法"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := newReportFixture()
			rp, err := svc.Create(context.Background(), tt.reporterID, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rp.Status != constants.ReportStatusPending {
				t.Fatalf("expected pending status, got %s", rp.Status)
			}
			if rp.ProductID == 0 {
				t.Fatalf("expected resolved product id")
			}
		})
	}
}

func TestReportServiceCreateKeepsSinglePending(t *testing.T) {
	svc, reports, _ := newReportFixture()
	req := &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 1, Reason: constants.ReportReasonFake, Description: "第一次举报"}
	if _, err := svc.Create(context.Background(), 1, req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := svc.Create(context.Background(), 1, req); err == nil {
		t.Fatalf("expected duplicate pending report conflict")
	}
	if _, err := svc.Create(context.Background(), 2, req); err != nil {
		t.Fatalf("other reporter should be allowed: %v", err)
	}
	// after the first report is rejected, the same reporter can report again
	if _, err := svc.Handle(context.Background(), 100, 1, &dto.HandleReportRequest{Action: constants.ReportActionReject, Result: "证据不足"}); err != nil {
		t.Fatalf("reject failed: %v", err)
	}
	if _, err := svc.Create(context.Background(), 1, req); err != nil {
		t.Fatalf("re-report after rejection should be allowed: %v", err)
	}
	var pending int
	for _, rp := range reports.reports {
		if rp.ReporterID == 1 && rp.TargetID == 1 && rp.Status == constants.ReportStatusPending {
			pending++
		}
	}
	if pending != 1 {
		t.Fatalf("expected exactly one pending report, got %d", pending)
	}
}

func TestReportServiceHandle(t *testing.T) {
	newPendingReport := func(t *testing.T, svc *ReportService) *model.Report {
		t.Helper()
		rp, err := svc.Create(context.Background(), 1, &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 1, Reason: constants.ReportReasonProhibited, Description: "疑似违禁品"})
		if err != nil {
			t.Fatalf("create report failed: %v", err)
		}
		return rp
	}

	t.Run("reject records outcome", func(t *testing.T) {
		svc, _, products := newReportFixture()
		rp := newPendingReport(t, svc)
		handled, err := svc.Handle(context.Background(), 100, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionReject, Result: "证据不足"})
		if err != nil {
			t.Fatalf("reject failed: %v", err)
		}
		if handled.Status != constants.ReportStatusRejected {
			t.Fatalf("expected rejected status, got %s", handled.Status)
		}
		if handled.HandledAt == nil || handled.HandledBy != 100 || handled.Result != "证据不足" {
			t.Fatalf("expected handled metadata recorded, got %+v", handled)
		}
		if products.products[1].Status != constants.ProductStatusOnSale {
			t.Fatalf("reject must not change product status")
		}
	})

	t.Run("takedown removes product", func(t *testing.T) {
		svc, _, products := newReportFixture()
		rp := newPendingReport(t, svc)
		handled, err := svc.Handle(context.Background(), 100, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionTakedown, Result: "违规商品已下架"})
		if err != nil {
			t.Fatalf("takedown failed: %v", err)
		}
		if handled.Status != constants.ReportStatusResolved {
			t.Fatalf("expected resolved status, got %s", handled.Status)
		}
		if handled.HandledAt == nil {
			t.Fatalf("expected handled_at recorded")
		}
		if products.products[1].Status != constants.ProductStatusRemoved {
			t.Fatalf("expected product removed, got %s", products.products[1].Status)
		}
	})

	t.Run("handle twice conflicts", func(t *testing.T) {
		svc, _, _ := newReportFixture()
		rp := newPendingReport(t, svc)
		if _, err := svc.Handle(context.Background(), 100, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionReject}); err != nil {
			t.Fatalf("first handle failed: %v", err)
		}
		if _, err := svc.Handle(context.Background(), 100, rp.ID, &dto.HandleReportRequest{Action: constants.ReportActionTakedown}); err == nil {
			t.Fatalf("expected conflict on already handled report")
		}
	})

	t.Run("invalid action rejected", func(t *testing.T) {
		svc, _, _ := newReportFixture()
		rp := newPendingReport(t, svc)
		if _, err := svc.Handle(context.Background(), 100, rp.ID, &dto.HandleReportRequest{Action: "explode"}); err == nil {
			t.Fatalf("expected invalid action error")
		}
	})

	t.Run("missing report not found", func(t *testing.T) {
		svc, _, _ := newReportFixture()
		if _, err := svc.Handle(context.Background(), 100, 999, &dto.HandleReportRequest{Action: constants.ReportActionReject}); err == nil {
			t.Fatalf("expected not found error")
		}
	})
}

func TestReportServiceList(t *testing.T) {
	svc, _, _ := newReportFixture()
	if _, err := svc.Create(context.Background(), 1, &dto.CreateReportRequest{TargetType: constants.ReportTargetProduct, TargetID: 1, Reason: constants.ReportReasonFake, Description: "假货"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	result, err := svc.List(context.Background(), &dto.ListReportQuery{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one pending report, got %d", result.Total)
	}
	if _, err := svc.List(context.Background(), &dto.ListReportQuery{Status: "nonsense"}); err == nil {
		t.Fatalf("expected invalid status error")
	}
	done, err := svc.List(context.Background(), &dto.ListReportQuery{Status: constants.ReportStatusRejected})
	if err != nil {
		t.Fatalf("list rejected failed: %v", err)
	}
	if done.Total != 0 {
		t.Fatalf("expected zero rejected reports, got %d", done.Total)
	}
}
