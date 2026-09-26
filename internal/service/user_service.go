package service

import "github.com/harkandavez/split-bill-go/internal/model/dto"

type UserService interface {
	AuthAnon(anonID string) (res dto.AnonRs, err error)
}
