package service

import (
	"context"
	"errors"
	"strings"
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
	repo           repository.UserRepository
	jwtSecret      string
	googleClientID string
	googleVerifier GoogleTokenVerifier
}

func NewUserService(
	repo repository.UserRepository,
	jwtSecret string,
	googleClientID string,
	googleVerifier GoogleTokenVerifier,
) UserService {
	return &userServiceImpl{
		repo:           repo,
		jwtSecret:      jwtSecret,
		googleClientID: googleClientID,
		googleVerifier: googleVerifier,
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

func profileOf(user entity.User) dto.UserProfileRs {
	image := ""
	if user.Image != nil {
		image = *user.Image
	}
	provider := "anonymous"
	if strings.EqualFold(user.OAuthProvider, "google") && user.OAuthID != "" {
		provider = "google"
	}
	return dto.UserProfileRs{
		UserID:    user.ID,
		Name:      user.Username,
		Email:     user.Email,
		Image:     image,
		Role:      user.Role,
		Provider:  provider,
		IsPremium: strings.EqualFold(strings.TrimSpace(user.Role), "PREMIUM"),
	}
}

func applyGoogleProfile(user entity.User, identity GoogleIdentity) entity.User {
	user.OAuthProvider = "google"
	user.OAuthID = identity.Sub
	user.Email = identity.Email
	if user.EmailVerifiedAt == nil {
		now := time.Now()
		user.EmailVerifiedAt = &now
	}
	if user.Role == "" || strings.EqualFold(user.Role, "ANONYMOUS") {
		user.Role = "USER"
	}
	name := strings.TrimSpace(identity.Name)
	if name == "" {
		name = user.Username
	}
	if name == "" && identity.Email != "" {
		name = strings.Split(identity.Email, "@")[0]
	}
	if name != "" {
		if len(name) > 100 {
			name = name[:100]
		}
		user.Username = name
	}
	if identity.Picture != "" {
		picture := identity.Picture
		user.Image = &picture
	}
	return user
}

// AuthGoogle menukar ID token Google dengan JWT aplikasi.
// currentUserID = user anon di device ini (untuk upgrade, bukan merge).
func (u *userServiceImpl) AuthGoogle(
	ctx context.Context,
	idToken string,
	currentUserID *int64,
) (dto.AuthUserRs, error) {
	if strings.TrimSpace(idToken) == "" {
		return dto.AuthUserRs{}, model.NewBadRequest("id_token wajib diisi")
	}
	if strings.TrimSpace(u.googleClientID) == "" {
		return dto.AuthUserRs{}, model.NewNotImplemented("Google login belum dikonfigurasi")
	}
	if u.googleVerifier == nil {
		return dto.AuthUserRs{}, model.NewInternalError("Google verifier tidak tersedia")
	}

	identity, err := u.googleVerifier.Verify(ctx, idToken, strings.TrimSpace(u.googleClientID))
	if err != nil {
		return dto.AuthUserRs{}, model.NewBadRequest("Google token tidak valid")
	}
	if identity.Email == "" || !identity.EmailVerified {
		return dto.AuthUserRs{}, model.NewBadRequest("Email Google belum terverifikasi")
	}

	var user entity.User
	found := false

	// 1) Akun Google sudah pernah login.
	user, err = u.repo.FindByOAuth("google", identity.Sub)
	if err == nil {
		found = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.AuthUserRs{}, model.NewInternalError("Database error")
	}

	// 2) Email sudah terdaftar (akun lama / provider lain).
	if !found {
		user, err = u.repo.FindByEmail(identity.Email)
		if err == nil {
			found = true
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AuthUserRs{}, model.NewInternalError("Database error")
		}
	}

	// 3) Upgrade user anonymous di device ini (tanpa merge).
	if !found && currentUserID != nil {
		anon, err := u.repo.FindByID(*currentUserID)
		if err == nil {
			// Jangan bajak akun yang sudah tertaut ke Google lain.
			if anon.OAuthID == "" || anon.OAuthID == identity.Sub {
				user = anon
				found = true
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AuthUserRs{}, model.NewInternalError("Database error")
		}
	}

	if found {
		user = applyGoogleProfile(user, identity)
		if err := u.repo.Save(user); err != nil {
			return dto.AuthUserRs{}, model.NewInternalError("Gagal menyimpan akun Google")
		}
	} else {
		user = applyGoogleProfile(entity.User{}, identity)
		if err := u.repo.Save(user); err != nil {
			return dto.AuthUserRs{}, model.NewInternalError("Gagal membuat akun Google")
		}
	}

	accessToken, err := u.generateAccessToken(dto.JWTPayload{UserID: user.ID, Role: user.Role})
	if err != nil {
		return dto.AuthUserRs{}, model.NewInternalError(err.Error())
	}

	return dto.AuthUserRs{
		AccessToken:   accessToken,
		UserProfileRs: profileOf(user),
	}, nil
}

func (u *userServiceImpl) Me(userID int64) (dto.UserProfileRs, error) {
	user, err := u.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.UserProfileRs{}, model.NewNotFound("User tidak ditemukan")
		}
		return dto.UserProfileRs{}, model.NewInternalError("Database error")
	}
	return profileOf(user), nil
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
