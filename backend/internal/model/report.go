package model

import "time"

// Report is a user complaint filed against a product or a completed trade order.
type Report struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ReporterID  uint       `gorm:"index;not null" json:"reporter_id"`
	TargetType  string     `gorm:"size:16;index;not null" json:"target_type"`
	TargetID    uint       `gorm:"not null" json:"target_id"`
	ProductID   uint       `gorm:"index;not null" json:"product_id"`
	Reason      string     `gorm:"size:32;not null" json:"reason"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	Result      string     `gorm:"size:255" json:"result"`
	HandledBy   uint       `json:"handled_by"`
	HandledAt   *time.Time `json:"handled_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
