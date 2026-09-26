package entity

import (
	"time"

	"gorm.io/datatypes"
)

// Opsional: go get gorm.io/datatypes

type ActivityLog struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	BillID        int64           `gorm:"column:bill_id;not null"  json:"bill_id"`
	ParticipantID int64           `gorm:"column:participant_id;not null" json:"participant_id"`
	ActionType    string          `gorm:"column:action_type;type:varchar(50);not null" json:"action_type"`
	TargetName    *string         `gorm:"column:target_name;type:varchar(255)" json:"target_name"`
	Amount        *int64          `gorm:"column:amount;type:numeric" json:"amount"`
	Metadata      *datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

// TableName menentukan nama tabel secara eksplisit (best practice di GORM)
func (ActivityLog) TableName() string {
	return "activity_logs"
}
