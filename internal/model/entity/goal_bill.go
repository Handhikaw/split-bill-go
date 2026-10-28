package entity

import "time"

type GoalBill struct {
	ID           int64     `gorm:"primaryKey;column:id"`
	BillID       int64     `gorm:"column:bill_id;not null;unique"` // 1:1 dengan bill
	TargetAmount int64     `gorm:"column:target_amount;not null"`
	Description  *string   `gorm:"column:description;size:255"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GoalBill) TableName() string {
	return "goal_bill"
}
