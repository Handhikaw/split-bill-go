package entity

import "time"

type User struct {
	ID              int64      `gorm:"primaryKey;column:id"`
	Username        string     `gorm:"column:username;size:100"`
	Email           string     `gorm:"column:email;size:150;unique"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	Image           *string    `gorm:"column:image"`
	Role            string     `gorm:"column:role;size:50"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime;<-:create"`
	AnonId          string     `gorm:"column:anon_id"`
	OAuthProvider   string     `gorm:"column:oauth_provider"`
	OAuthID         string     `gorm:"column:oauth_id"`
}

func (User) TableName() string {
	return "users"
}
