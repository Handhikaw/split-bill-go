package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type userRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) FindByID(id int64) (entity.User, error) {
	var user entity.User
	err := r.db.Where("id = ?", id).First(&user).Error
	return user, err
}

func (r *userRepositoryImpl) FindByEmail(email string) (entity.User, error) {
	var user entity.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *userRepositoryImpl) FindByAnonID(anonID string) (entity.User, error) {
	var user entity.User
	err := r.db.Where("anon_id = ?", anonID).First(&user).Error
	return user, err
}

func (r *userRepositoryImpl) FindByOAuth(provider, oauthID string) (entity.User, error) {
	var user entity.User
	err := r.db.Where("oauth_provider = ? AND oauth_id = ?", provider, oauthID).First(&user).Error
	return user, err
}

func (r *userRepositoryImpl) Save(user entity.User) error {
	return r.db.Save(&user).Error
}
