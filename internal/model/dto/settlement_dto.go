package dto

import "time"

type SettlementPaymentRq struct {
	FromParticipantID int64   `json:"from_participant_id" binding:"required"`
	ToParticipantID   int64   `json:"to_participant_id" binding:"required"`
	Amount            string  `json:"amount" binding:"required"`
	Note              *string `json:"note"`
}

type SettlementPaymentItem struct {
	ID     int64      `json:"id"`
	From   PersonInfo `json:"from"`
	To     PersonInfo `json:"to"`
	Amount string     `json:"amount"`
	Note   *string    `json:"note"`
	PaidAt time.Time  `json:"paid_at"`
}
