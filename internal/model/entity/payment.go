package entity

import (
	"time"
)

type Payment struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	BillID    int64     `gorm:"column:bill_id;not null"`
	ExpenseID int64     `gorm:"column:expense_id;not null"`
	PaidBy    int64     `gorm:"column:paid_by;not null"` //ParticipantID
	Amount    int64     `gorm:"column:amount;not null"`
	PaidAt    time.Time `gorm:"column:paid_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Payment) TableName() string {
	return "payment"
}
