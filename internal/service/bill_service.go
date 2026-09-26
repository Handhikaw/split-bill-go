package service

import (
	"github.com/harkandavez/split-bill-go/internal/model/dto"
)

type BillService interface {
	CreateSplitBill(req dto.CreateSpiltBillRq, actorUserID *int64) (dto.CreateSpiltBillRs, error)
	GetSplitBill(publicID string) (*dto.GetSplitBillRs, error)
	JoinSplitBill(publicID string, req dto.JoinSplitBillRq) error
	CreateGroupBill(req dto.CreateGroupBillRq, actorUserID *int64) (dto.CreateGroupBillRs, error)
	GetGroupBill(publicID string) (dto.GetGroupBillRs, error)
}
