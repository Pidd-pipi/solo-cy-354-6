package dto

// CreateReportRequest is the payload for filing a report against a product or trade order.
type CreateReportRequest struct {
	TargetType  string `json:"target_type" binding:"required,oneof=product trade_order"`
	TargetID    uint   `json:"target_id" binding:"required"`
	Reason      string `json:"reason" binding:"required,max=32"`
	Description string `json:"description" binding:"required,max=500"`
}

// HandleReportRequest is the admin payload for rejecting or resolving a report.
type HandleReportRequest struct {
	Action string `json:"action" binding:"required,oneof=reject takedown"`
	Result string `json:"result" binding:"max=255"`
}

// ListReportQuery filters the admin report list.
type ListReportQuery struct {
	PageQuery
	Status string `form:"status" json:"status"`
}
