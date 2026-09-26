package dto

import "time"

type ActivityItem struct {
	ID         int64     `json:"id"`
	ActorName  string    `json:"actor_name"`
	ActionType string    `json:"action_type"`
	TargetName *string   `json:"target_name"`
	Amount     *string   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
}
