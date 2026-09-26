package entity

import (
	"time"
)

type ExpenseParticipant struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	ExpenseID     int64     `gorm:"column:expense_id;not null"`
	ParticipantID int64     `gorm:"column:participant_id;not null"`
	ShareAmount   int64     `gorm:"column:share_amount;not null"`
	ShareType     string    `gorm:"column:share_type;size:20;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ExpenseParticipant) TableName() string {
	return "expense_participant"
}
