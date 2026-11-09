package service

import "github.com/harkandavez/split-bill-go/internal/model/dto"

type ParticipantService interface {
	UpsertParticipant(publicID string, actionParticipant int64, authUserID *int64, req dto.ParticipantRq) (dto.ParticipantRs, error)
	RemoveParticipant(publicID string, id int64, actionParticipant int64) error
}
