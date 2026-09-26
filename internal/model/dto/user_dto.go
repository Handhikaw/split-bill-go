package dto

type AnonRs struct {
	AccessToken string `json:"access_token"`
	UserID      int64  `json:"user_id"`
	Role        string `json:"role"`
}

type JWTPayload struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}
