package service

import (
	"errors"

	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

// lookupUserByID mengambil role/username user pembuat bill.
// Nil = guest (tanpa token), bukan error.
func lookupUserByID(db *gorm.DB, userID *int64) (entity.User, error) {
	var user entity.User
	if userID == nil {
		return user, nil
	}
	if err := db.Select("id, role, username").Where("id = ?", *userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, model.NewNotFound("user not found")
		}
		return entity.User{}, model.NewInternalError("Database error")
	}
	return user, nil
}
