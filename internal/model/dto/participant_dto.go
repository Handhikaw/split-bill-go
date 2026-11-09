package dto

type ParticipantRq struct {
	ID      *int64 `json:"id"`
	Name    string `json:"name" binding:"required"`
	UserID  *int64 `json:"user_id"`
	IsOwner *bool  `json:"is_owner"`
}

type ParticipantRs struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ParticipantBalance struct {
	ParticipantID int64  `json:"participant_id"`
	Name          string `json:"name"`
	IsOwner       bool   `json:"is_owner"`
	TotalPaid     string `json:"total_paid"`  // Total yang dia bayarkan (Credit)
	TotalOwed     string `json:"total_owed"`  // Total utang dari item yg dimakan (Debit)
	NetBalance    string `json:"net_balance"` // Paid - Owed
}
