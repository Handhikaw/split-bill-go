package dto

import "time"

type CreateSpiltBillRq struct {
	Title       string  `json:"title" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	CreatorName *string `json:"creator_name"`
}

type CreateSpiltBillRs struct {
	PublicId           string `json:"public_id"`
	OwnerParticipantID int64  `json:"owner_participant_id"`
}

type CreateGroupBillRq struct {
	Title       string  `json:"title" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	CreatorName *string `json:"creator_name"`
}

type CreateGroupBillRs struct {
	PublicId           string     `json:"public_id"`
	Title              string     `json:"title"`
	Status             string     `json:"status"`
	ExpiredAt          *time.Time `json:"expired_at"`
	OwnerParticipantID int64      `json:"owner_participant_id"`
}

type JoinSplitBillRq struct {
	Name   string `json:"name" binding:"required"`
	UserID *int64 `json:"user_id"`
}

type GetSplitBillRs struct {
	ID            int64                `json:"id"`
	ExpenseID     int64                `json:"expense_id"`
	Status        string               `json:"status"`
	Name          string               `json:"name"`
	Currency      string               `json:"currency"`
	TotalAmount   string               `json:"total_amount"`
	Tax           string               `json:"tax"`
	ServiceCharge string               `json:"service_charge"`
	Discount      string               `json:"discount"`
	PayerID       int64                `json:"payer_id"`
	Items         []ExpenseItem        `json:"items"`
	Participants  []ParticipantBalance `json:"participants"`
	Settlements   []Settlement         `json:"settlements"`
}

type GetGroupBillRs struct {
	ID           int64                `json:"id"`
	Status       string               `json:"status"`
	Name         string               `json:"name"`
	Currency     string               `json:"currency"`
	TotalAmount  string               `json:"total_amount"`
	Plan         string               `json:"plan"`
	ReadOnly     bool                 `json:"read_only"`
	ExpiredAt    *time.Time           `json:"expired_at"`
	Limits       BillLimits           `json:"limits"`
	Usage        BillUsage            `json:"usage"`
	Expense      []Expense            `json:"expense"`
	Participants []ParticipantBalance `json:"participants"`
	Settlements  []Settlement         `json:"settlements"`
}

// BillLimits: 0 = unlimited (Premium).
type BillLimits struct {
	MaxParticipants int `json:"max_participants"`
	MaxExpenses     int `json:"max_expenses"`
}

type BillUsage struct {
	ParticipantCount int `json:"participant_count"`
	ExpenseCount     int `json:"expense_count"`
}

type Settlement struct {
	From   PersonInfo `json:"from"`
	To     PersonInfo `json:"to"`
	Amount string     `json:"amount"`
}

type PersonInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
