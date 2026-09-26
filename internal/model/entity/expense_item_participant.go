package entity

type ExpenseItemParticipant struct {
	ID            int64 `gorm:"primaryKey;column:id"`
	ItemID        int64 `gorm:"column:item_id;not null"`
	ParticipantID int64 `gorm:"column:participant_id;not null"`
	ShareAmount   int64 `gorm:"column:share_amount;not null"`
}

func (ExpenseItemParticipant) TableName() string {
	return "expense_item_participant"
}
