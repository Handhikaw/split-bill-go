package service

import "github.com/harkandavez/split-bill-go/internal/model/dto"

type SettlementService interface {
	Create(publicID string, actorParticipantID int64, req dto.SettlementPaymentRq) (dto.SettlementPaymentItem, error)
	Delete(publicID string, settlementID int64, actorParticipantID int64) error
}
