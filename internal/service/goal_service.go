package service

import "github.com/harkandavez/split-bill-go/internal/model/dto"

type GoalService interface {
	CreateGoalBill(req dto.CreateGoalBillRq, actorUserID *int64) (dto.CreateGoalBillRs, error)
	GetGoalBill(publicID string) (dto.GetGoalBillRs, error)
	AddContribution(publicID string, actorParticipantID int64, req dto.GoalContributionRq) (dto.GoalContributionItem, error)
	DeleteContribution(publicID string, contributionID int64, actorParticipantID int64) error
	UpdateGoalBill(publicID string, actorParticipantID int64, req dto.UpdateGoalBillRq) (dto.GetGoalBillRs, error)
}
