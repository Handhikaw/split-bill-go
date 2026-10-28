package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type GoalRepository interface {
	GetGoalByBillID(billID int64) (*entity.GoalBill, error)
	ListContributions(billID int64) ([]entity.Contribution, error)
	SumContributions(billID int64) (int64, error)
	CreateContribution(contribution *entity.Contribution) error
	FindContributionByIDAndBillID(id, billID int64) (*entity.Contribution, error)
	DeleteContribution(contribution *entity.Contribution) error
	CountContributionsByParticipantID(participantID int64) (int64, error)
	UpdateGoalBill(billID int64, billFields map[string]any, goalFields map[string]any) error
}
