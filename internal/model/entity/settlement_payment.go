package entity

import "time"

// SettlementPayment = transfer yang sudah dilakukan debitur ke kreditur.
// Satu baris mengurangi utang: from.net += amount, to.net -= amount.
type SettlementPayment struct {
	ID                int64     `gorm:"primaryKey;column:id"`
	BillID            int64     `gorm:"column:bill_id;not null"`
	FromParticipantID int64     `gorm:"column:from_participant_id;not null"`
	ToParticipantID   int64     `gorm:"column:to_participant_id;not null"`
	Amount            int64     `gorm:"column:amount;not null"`
	Note              *string   `gorm:"column:note;size:255"`
	PaidAt            time.Time `gorm:"column:paid_at"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime;<-:create"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SettlementPayment) TableName() string {
	return "settlement_payment"
}
