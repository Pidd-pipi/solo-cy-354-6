package constants

// ReportTargetType defines the kinds of objects a report can point at.
const (
	ReportTargetProduct    = "product"
	ReportTargetTradeOrder = "trade_order"
)

// ReportTargetTypes lists all valid report target types.
var ReportTargetTypes = []string{
	ReportTargetProduct, ReportTargetTradeOrder,
}

// IsReportTargetType reports whether the given target type is valid.
func IsReportTargetType(t string) bool {
	for _, v := range ReportTargetTypes {
		if v == t {
			return true
		}
	}
	return false
}

// ReportStatus defines the report handling lifecycle shared with the frontend.
const (
	ReportStatusPending  = "pending"
	ReportStatusRejected = "rejected"
	ReportStatusResolved = "resolved"
)

// ReportStatuses lists all valid report statuses.
var ReportStatuses = []string{
	ReportStatusPending, ReportStatusRejected, ReportStatusResolved,
}

// IsReportStatus reports whether the given status is valid.
func IsReportStatus(s string) bool {
	for _, v := range ReportStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// Report reason enum values shared with the frontend.
const (
	ReportReasonFraud      = "fraud"
	ReportReasonFake       = "fake"
	ReportReasonProhibited = "prohibited"
	ReportReasonAbuse      = "abuse"
	ReportReasonOther      = "other"
)

// ReportReasons lists all valid report reasons.
var ReportReasons = []string{
	ReportReasonFraud, ReportReasonFake, ReportReasonProhibited, ReportReasonAbuse, ReportReasonOther,
}

// IsReportReason reports whether the given reason is valid.
func IsReportReason(r string) bool {
	for _, v := range ReportReasons {
		if v == r {
			return true
		}
	}
	return false
}

// Report handle action enum values shared with the frontend.
const (
	ReportActionReject   = "reject"
	ReportActionTakedown = "takedown"
)

// ReportTargetTypeText returns the Chinese label of a report target type.
func ReportTargetTypeText(t string) string {
	switch t {
	case ReportTargetProduct:
		return "商品"
	case ReportTargetTradeOrder:
		return "交易订单"
	default:
		return "未知"
	}
}

// ReportStatusText returns the Chinese label of a report status.
func ReportStatusText(s string) string {
	switch s {
	case ReportStatusPending:
		return "待处理"
	case ReportStatusRejected:
		return "已驳回"
	case ReportStatusResolved:
		return "已处理"
	default:
		return "未知"
	}
}

// ReportReasonText returns the Chinese label of a report reason.
func ReportReasonText(r string) string {
	switch r {
	case ReportReasonFraud:
		return "诈骗行为"
	case ReportReasonFake:
		return "虚假信息"
	case ReportReasonProhibited:
		return "违禁物品"
	case ReportReasonAbuse:
		return "辱骂骚扰"
	case ReportReasonOther:
		return "其他"
	default:
		return "未知"
	}
}
