package dto

import "time"

type CreateGoalBillRq struct {
	Title        string  `json:"title" binding:"required"`
	Currency     string  `json:"currency" binding:"required"`
	TargetAmount string  `json:"target_amount" binding:"required"`
	Description  *string `json:"description"`
	CreatorName  *string `json:"creator_name"`
}

type CreateGoalBillRs struct {
	PublicID           string     `json:"public_id"`
	Title              string     `json:"title"`
	Status             string     `json:"status"`
	TargetAmount       string     `json:"target_amount"`
	ExpiredAt          *time.Time `json:"expired_at"`
	OwnerParticipantID int64      `json:"owner_participant_id"`
}

type GoalContributionRq struct {
	ParticipantID int64   `json:"participant_id" binding:"required"`
	Amount        string  `json:"amount" binding:"required"`
	Note          *string `json:"note"`
}

// UpdateGoalBillRq: minimal satu field diisi. target_amount = string major
// sesuai kontrak money.FromString.
type UpdateGoalBillRq struct {
	Title        *string `json:"title"`
	TargetAmount *string `json:"target_amount"`
	Description  *string `json:"description"`
}

type GoalContributionItem struct {
	ID            int64     `json:"id"`
	ParticipantID int64     `json:"participant_id"`
	Name          string    `json:"name"`
	Amount        string    `json:"amount"`
	Note          *string   `json:"note"`
	PaidAt        time.Time `json:"paid_at"`
}

type GoalParticipantProgress struct {
	ParticipantID     int64  `json:"participant_id"`
	Name              string `json:"name"`
	IsOwner           bool   `json:"is_owner"`
	TotalContributed  string `json:"total_contributed"`
	ContributionCount int    `json:"contribution_count"`
}

type GetGoalBillRs struct {
	ID              int64                     `json:"id"`
	Status          string                    `json:"status"`
	Name            string                    `json:"name"`
	Currency        string                    `json:"currency"`
	TargetAmount    string                    `json:"target_amount"`
	TotalCollected  string                    `json:"total_collected"`
	Remaining       string                    `json:"remaining"`
	Description     *string                   `json:"description"`
	ExpiredAt       *time.Time                `json:"expired_at"`
	Plan            string                    `json:"plan"`
	ReadOnly        bool                      `json:"read_only"`
	ViewerIsPremium bool                      `json:"viewer_is_premium"`
	Limits          BillLimits                `json:"limits"`
	Usage           BillUsage                 `json:"usage"`
	Participants    []GoalParticipantProgress `json:"participants"`
	Contributions   []GoalContributionItem    `json:"contributions"`
}
