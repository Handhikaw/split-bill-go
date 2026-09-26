package entity

import "time"

type Participant struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	BillID    int64     `gorm:"column:bill_id;not null"`
	Name      string    `gorm:"column:name;size:100;not null"`
	UserID    *int64    `gorm:"column:user_id"`
	IsOwner   bool      `gorm:"column:is_owner;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
}

func (Participant) TableName() string {
	return "participant"
}
