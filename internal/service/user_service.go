package service

import (
	"context"

	"github.com/harkandavez/split-bill-go/internal/model/dto"
)

type UserService interface {
	AuthAnon(anonID string) (res dto.AnonRs, err error)
	AuthGoogle(ctx context.Context, idToken string, currentUserID *int64) (dto.AuthUserRs, error)
	Me(userID int64) (dto.UserProfileRs, error)
}
