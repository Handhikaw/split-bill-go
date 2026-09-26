package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type UserRepository interface {
	FindByEmail(email string) (entity.User, error)
	FindByAnonID(anonID string) (entity.User, error)
	Save(user entity.User) error
}
