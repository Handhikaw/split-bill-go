package entity

import "time"

// Contribution = uang yang sudah diterima untuk patungan.
// Satu baris bukan "janji"; tidak ada status pledge.
type Contribution struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	BillID        int64     `gorm:"column:bill_id;not null"`
	ParticipantID int64     `gorm:"column:participant_id;not null"`
	Amount        int64     `gorm:"column:amount;not null"`
	Note          *string   `gorm:"column:note;size:255"`
	PaidAt        time.Time `gorm:"column:paid_at"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Contribution) TableName() string {
	return "contribution"
}
