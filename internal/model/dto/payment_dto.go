package dto

type PaymentRq struct {
	ExpenseID     int64  `json:"expense_id" binding:"required"`
	ParticipantID int64  `json:"participant_id" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	// Opsional: payer lama yang diganti. Bila diisi dan beda dari
	// ParticipantID, payment lama dihapus dalam transaksi yang sama
	// (pindah payer, bukan nambah baris).
	OldParticipantID *int64 `json:"old_participant_id"`
}

type PaymentItemRq struct {
	ParticipantID int64  `json:"participant_id"`
	Amount        string `json:"amount"`
}

// ReplacePaymentsRq mengganti seluruh payer satu expense (multi-payer).
type ReplacePaymentsRq struct {
	ExpenseID int64           `json:"expense_id" binding:"required"`
	Payments  []PaymentItemRq `json:"payments"`
}
