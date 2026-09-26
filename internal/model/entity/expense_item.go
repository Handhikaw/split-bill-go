package entity

import (
	"time"
)

type ExpenseItem struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	ExpenseID int64     `gorm:"column:expense_id;not null"`
	Name      string    `gorm:"column:name;size:150;not null"`
	Quantity  int       `gorm:"column:quantity;not null"`
	Price     int64     `gorm:"column:price;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ExpenseItem) TableName() string {
	return "expense_item"
}
