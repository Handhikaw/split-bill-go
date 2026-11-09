package dto

import "time"

type MyBillItem struct {
	PublicID    string `json:"public_id"`
	Kind        string `json:"kind"` // split | group
	Title       string `json:"title"`
	Currency    string `json:"currency"`
	TotalAmount string `json:"total_amount"`
	Status      string `json:"status"`
	// Net balance saya di bill ini. Null bila saya bukan peserta
	// (mis. bill yang saya buat tapi belum join) atau bill patungan.
	MyNet       *string `json:"my_net"`
	MemberCount int     `json:"member_count"`
	// Khusus patungan: target dan total terkumpul. Null untuk split/group.
	TargetAmount    *string    `json:"target_amount,omitempty"`
	CollectedAmount *string    `json:"collected_amount,omitempty"`
	Plan            string     `json:"plan"`
	ReadOnly        bool       `json:"read_only"`
	ExpiredAt       *time.Time `json:"expired_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
