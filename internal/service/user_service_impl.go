package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type userServiceImpl struct {
	repo      repository.UserRepository
	jwtSecret string
}

func NewUserService(
	repo repository.UserRepository,
	jwtSecret string,
) UserService {
	return &userServiceImpl{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// AuthAnon implements [UserService].
func (u *userServiceImpl) AuthAnon(anonID string) (res dto.AnonRs, err error) {
	if anonID == "" {
		return res, model.NewBadRequest("X-Anonymous-Id header is required")
	}
	if _, err := uuid.Parse(anonID); err != nil {
		return res, model.NewBadRequest("X-Anonymous-Id must be a valid UUID")
	}

	user, err := u.repo.FindByAnonID(anonID)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return res, model.NewInternalError(err.Error())
		}

		user = entity.User{
			AnonId: anonID,
			Role:   "ANONYMOUS",
		}

		if err := u.repo.Save(user); err != nil {
			return res, model.NewInternalError(err.Error())
		}

		user, err = u.repo.FindByAnonID(anonID)
		if err != nil {
			return res, model.NewInternalError(err.Error())
		}
	}

	accessToken, err := u.generateAccessToken(dto.JWTPayload{
		UserID: user.ID,
		Role:   user.Role,
	})
	if err != nil {
		return res, model.NewInternalError(err.Error())
	}

	return dto.AnonRs{
		AccessToken: accessToken,
		UserID:      user.ID,
		Role:        user.Role,
	}, nil
}

func (u *userServiceImpl) generateAccessToken(payload dto.JWTPayload) (string, error) {
	if u.jwtSecret == "" {
		return "", model.NewInternalError("JWT_SECRET is not configured")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": payload.UserID,
		"role":    payload.Role,
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})

	return token.SignedString([]byte(u.jwtSecret))
}
