package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/dto"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// ReportRepository is the data access contract for report rows.
type ReportRepository interface {
	Transaction(ctx context.Context, fn func(txCtx context.Context) error) error
	Create(ctx context.Context, rp *model.Report) error
	FindByID(ctx context.Context, id uint) (*model.Report, error)
	FindPendingByReporterAndTarget(ctx context.Context, reporterID uint, targetType string, targetID uint) (*model.Report, error)
	List(ctx context.Context, status string, page, pageSize int) ([]model.Report, int64, error)
	MarkHandled(ctx context.Context, id uint, status, result string, handledBy uint, handledAt time.Time) error
}

// ReportProductRepository is the product data contract used by ReportService.
type ReportProductRepository interface {
	FindByID(ctx context.Context, id uint) (*model.Product, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
}

// ReportOrderRepository is the trade order data contract used by ReportService.
type ReportOrderRepository interface {
	FindByID(ctx context.Context, id uint) (*model.TradeOrder, error)
}

// ReportService manages user reports and admin handling.
type ReportService struct {
	reports  ReportRepository
	products ReportProductRepository
	orders   ReportOrderRepository
	logger   *slog.Logger
}

// NewReportService wires the report service dependencies.
func NewReportService(reports ReportRepository, products ReportProductRepository, orders ReportOrderRepository, logger *slog.Logger) *ReportService {
	return &ReportService{reports: reports, products: products, orders: orders, logger: logger}
}

// Create files a report against a product or a completed trade order. Only one
// pending report is kept per reporter and target.
func (s *ReportService) Create(ctx context.Context, reporterID uint, req *dto.CreateReportRequest) (*model.Report, error) {
	if !constants.IsReportTargetType(req.TargetType) {
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportTarget, nil)
	}
	if !constants.IsReportReason(req.Reason) {
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportReasonInvalid, nil)
	}
	productID, err := s.resolveTarget(ctx, reporterID, req)
	if err != nil {
		return nil, err
	}
	rp := &model.Report{
		ReporterID: reporterID, TargetType: req.TargetType, TargetID: req.TargetID,
		ProductID: productID, Reason: req.Reason, Description: req.Description,
		Status: constants.ReportStatusPending,
	}
	if err := s.reports.Transaction(ctx, func(txCtx context.Context) error {
		if _, err := s.reports.FindPendingByReporterAndTarget(txCtx, reporterID, req.TargetType, req.TargetID); err == nil {
			return util.ErrConflict
		} else if !errors.Is(err, util.ErrNotFound) {
			return err
		}
		return s.reports.Create(txCtx, rp)
	}); err != nil {
		if errors.Is(err, util.ErrConflict) {
			return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportDuplicate, nil)
		}
		s.logger.Error(fmt.Sprintf(constants.LogReportCreateFailed, reporterID, req.TargetType, req.TargetID, err))
		return nil, util.WrapAppError(fmt.Errorf("report[reporter=%d] create: %w", reporterID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	s.logger.Info(fmt.Sprintf(constants.LogReportCreateSuccess, rp.ID, reporterID, req.TargetType, req.TargetID))
	return rp, nil
}

// resolveTarget validates the report target and returns the related product id.
func (s *ReportService) resolveTarget(ctx context.Context, reporterID uint, req *dto.CreateReportRequest) (uint, error) {
	switch req.TargetType {
	case constants.ReportTargetProduct:
		p, err := s.products.FindByID(ctx, req.TargetID)
		if err != nil {
			return 0, util.WrapAppError(fmt.Errorf("report[target=product:%d] lookup: %w", req.TargetID, err), 404, constants.CodeNotFound, constants.MsgReportTarget)
		}
		return p.ID, nil
	case constants.ReportTargetTradeOrder:
		order, err := s.orders.FindByID(ctx, req.TargetID)
		if err != nil {
			return 0, util.WrapAppError(fmt.Errorf("report[target=trade_order:%d] lookup: %w", req.TargetID, err), 404, constants.CodeNotFound, constants.MsgReportTarget)
		}
		if order.Status != constants.TradeStatusCompleted {
			return 0, util.NewAppError(409, constants.CodeConflict, "仅已完成的订单可以举报", nil)
		}
		if order.BuyerID != reporterID && order.SellerID != reporterID {
			return 0, util.NewAppError(403, constants.CodeForbidden, constants.MsgNotParticipant, nil)
		}
		return order.ProductID, nil
	default:
		return 0, util.NewAppError(400, constants.CodeValidation, constants.MsgReportTarget, nil)
	}
}

// List returns reports filtered by status for admins.
func (s *ReportService) List(ctx context.Context, q *dto.ListReportQuery) (*dto.PageResult, error) {
	q.Normalize()
	status := q.Status
	if status == "" {
		status = constants.ReportStatusPending
	}
	if !constants.IsReportStatus(status) {
		return nil, util.NewAppError(400, constants.CodeValidation, "举报状态不合法", nil)
	}
	items, total, err := s.reports.List(ctx, status, q.Page, q.PageSize)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report list: %w", err), 500, constants.CodeInternalError, constants.MsgInternalError)
	}
	return &dto.PageResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// Handle lets an admin reject a report or take the reported product down.
func (s *ReportService) Handle(ctx context.Context, adminID, reportID uint, req *dto.HandleReportRequest) (*model.Report, error) {
	rp, err := s.reports.FindByID(ctx, reportID)
	if err != nil {
		return nil, util.WrapAppError(fmt.Errorf("report[id=%d] handle find: %w", reportID, err), 404, constants.CodeNotFound, constants.MsgNotFound)
	}
	if rp.Status != constants.ReportStatusPending {
		return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportAlreadyHandled, nil)
	}
	now := time.Now()
	switch req.Action {
	case constants.ReportActionReject:
		if err := s.reports.MarkHandled(ctx, reportID, constants.ReportStatusRejected, req.Result, adminID, now); err != nil {
			if errors.Is(err, util.ErrConflict) {
				return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportAlreadyHandled, nil)
			}
			return nil, util.WrapAppError(fmt.Errorf("report[id=%d] reject: %w", reportID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
		rp.Status = constants.ReportStatusRejected
		s.logger.Info(fmt.Sprintf(constants.LogReportRejectSuccess, reportID, adminID))
	case constants.ReportActionTakedown:
		if err := s.reports.Transaction(ctx, func(txCtx context.Context) error {
			if err := s.reports.MarkHandled(txCtx, reportID, constants.ReportStatusResolved, req.Result, adminID, now); err != nil {
				return err
			}
			return s.products.UpdateStatus(txCtx, rp.ProductID, constants.ProductStatusRemoved)
		}); err != nil {
			if errors.Is(err, util.ErrConflict) {
				return nil, util.NewAppError(409, constants.CodeConflict, constants.MsgReportAlreadyHandled, nil)
			}
			return nil, util.WrapAppError(fmt.Errorf("report[id=%d] takedown: %w", reportID, err), 500, constants.CodeInternalError, constants.MsgInternalError)
		}
		rp.Status = constants.ReportStatusResolved
		s.logger.Info(fmt.Sprintf(constants.LogReportTakedownSuccess, reportID, rp.ProductID, adminID))
	default:
		return nil, util.NewAppError(400, constants.CodeValidation, constants.MsgReportActionInvalid, nil)
	}
	s.logger.Info(fmt.Sprintf(constants.LogReportHandleSuccess, reportID, req.Action))
	rp.Result = req.Result
	rp.HandledBy = adminID
	rp.HandledAt = &now
	return rp, nil
}
