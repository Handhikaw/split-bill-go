package entity

import (
	"time"
)

type Expense struct {
	ID         int64     `gorm:"primaryKey;column:id"`
	BillID     int64     `gorm:"column:bill_id;not null"`
	Title      string    `gorm:"column:title;size:150;not null"`
	Type       string    `gorm:"column:type;size:30;not null"`
	Amount     int64     `gorm:"column:amount"`
	Tax        int64     `gorm:"column:tax"`
	Discount   int64     `gorm:"column:discount"`
	Charge     int64     `gorm:"column:charge"`
	CategoryID *int64    `gorm:"column:category_id"`
	CreatedBy  *int64    `gorm:"column:created_by"` //participantId
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Expense) TableName() string {
	return "expense"
}
