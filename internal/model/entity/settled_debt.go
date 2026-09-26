package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type SettledDebt struct {
	ID                int64           `gorm:"primaryKey;column:id"`
	BillID            int64           `gorm:"column:bill_id;not null"`
	ExpenseID         *int64          `gorm:"column:expense_id"`
	FromParticipantID int64           `gorm:"column:from_participant_id;not null"`
	ToParticipantID   int64           `gorm:"column:to_participant_id;not null"`
	Amount            decimal.Decimal `gorm:"column:amount;type:numeric(12,2);not null"`
	Status            string          `gorm:"column:status;size:20;not null"`
	SettledAt         *time.Time      `gorm:"column:settled_at"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (SettledDebt) TableName() string {
	return "settled_debt"
}
