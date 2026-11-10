package dto

import "time"

type CreateExpenseRq struct {
	PublicID     string  `json:"public_id" validate:"required"`
	Title        string  `json:"title" validate:"required,max=255"`
	Type         string  `json:"type" validate:"required,max=30"`
	Amount       *string `json:"amount"`
	CategoryID   *int64  `json:"category"`
	CreatedBy    int64   `json:"created_by"`
	Participants []int64 `json:"participants" validate:"required,min=1,dive,gt=0"`
}

type CreateExpenseRs struct {
	ID           int64                `json:"id"`
	Title        string               `json:"title"`
	Type         string               `json:"type"`
	Amount       string               `json:"amount"`
	CategoryID   *int64               `json:"category"`
	CreatedBy    *int64               `json:"created_by"`
	Participants []ExpenseParticipant `json:"participants"`
}

type UpdateExpenseRq struct {
	Title        *string `json:"title"`
	Amount       *string `json:"amount"`
	Tax          *string `json:"tax"`
	Discount     *string `json:"discount"`
	Charge       *string `json:"charge"`
	CategoryID   *int64  `json:"category"`
	CreatedBy    *int64  `json:"created_by"`
	Participants []int64 `json:"participants"`
}

type UpdateExpenseRs struct {
	Title        string               `json:"title"`
	Type         string               `json:"type"`
	Amount       string               `json:"amount"`
	Tax          string               `json:"tax"`
	Discount     string               `json:"discount"`
	Charge       string               `json:"charge"`
	CategoryID   *int64               `json:"category"`
	CreatedBy    *int64               `json:"created_by"`
	Participants []ExpenseParticipant `json:"participants"`
}

type CreateExpenseItemRq struct {
	Name         string  `json:"name" binding:"required"`
	Price        string  `json:"price" binding:"required"`
	Quantity     int     `json:"quantity" binding:"required"`
	Participants []int64 `json:"participants"`
}

type CreateExpenseItemRs struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Price        string  `json:"price"`
	Quantity     int     `json:"quantity"`
	Total        string  `json:"total"`
	Participants []int64 `json:"participants"`
}

type UpdateExpenseItemRq struct {
	Name         *string `json:"name"`
	Price        *string `json:"price"`
	Quantity     *int    `json:"quantity"`
	Participants []int64 `json:"participants"` // Nil berarti tidak ganti orang, Empty [] berarti hapus semua
}

type ExpenseItemParticipant struct {
	ParticipantID int64  `json:"participant_id"`
	Name          string `json:"name"`
	ShareAmount   string `json:"share_amount"`
}

type ExpenseItem struct {
	ID           int64                    `json:"id"`
	Name         string                   `json:"name"`
	Price        string                   `json:"price"`
	Quantity     int                      `json:"quantity"`
	Subtotal     string                   `json:"subtotal"`
	Participants []ExpenseItemParticipant `json:"participants"`
}

type ExpenseParticipant struct {
	ParticipantID int64  `json:"participant_id"`
	ShareAmount   string `json:"share_amount"`
}

type ExpensePayment struct {
	ParticipantID int64  `json:"participant_id"`
	Name          string `json:"name"`
	Amount        string `json:"amount"`
}

type Expense struct {
	ID           int64                `json:"id"`
	Title        string               `json:"title"`
	Amount       string               `json:"amount"`
	Tax          string               `json:"tax"`
	Discount     string               `json:"discount"`
	Charge       string               `json:"charge"`
	Category     string               `json:"category"`   // category name
	Created_By   string               `json:"created_by"` // participant name
	Created_At   time.Time            `json:"created_at"`
	Participants []ExpenseParticipant `json:"participants"`
	Payments     []ExpensePayment     `json:"payments"`
	Items        *[]ExpenseItem       `json:"items"`
}

type CategoryRs struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	ColorCode string `json:"color_code"`
	IsDefault bool   `json:"is_default"`
}

type CategoryRq struct {
	ID        *int64  `json:"id"`
	UserID    int64   `json:"user_id"`
	Name      string  `json:"name"`
	Icon      *string `json:"icon"`
	ColorCode *string `json:"color_code"`
}
