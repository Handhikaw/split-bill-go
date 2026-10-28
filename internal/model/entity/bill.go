package entity

import "time"

type Bill struct {
	ID        int64      `gorm:"primaryKey;column:id"`
	PublicID  string     `gorm:"column:public_id;size:32;unique;not null"`
	Name      string     `gorm:"column:name;size:150;not null"`
	Currency  string     `gorm:"column:currency;size:3;not null"`
	Kind      string     `gorm:"column:kind;size:10;not null;default:group"` // split | group | goal
	Status    string     `gorm:"column:status;size:20;not null"`
	CreatedBy *int64     `gorm:"column:created_by"` //UserID
	ExpiredAt *time.Time `gorm:"column:expired_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (Bill) TableName() string {
	return "bill"
}
