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

type GoogleAuthRq struct {
	IDToken string `json:"id_token" binding:"required"`
}

type UserProfileRs struct {
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Image     string `json:"image"`
	Role      string `json:"role"`
	Provider  string `json:"provider"` // google | anonymous
	IsPremium bool   `json:"is_premium"`
}

type AuthUserRs struct {
	AccessToken string `json:"access_token"`
	UserProfileRs
}
