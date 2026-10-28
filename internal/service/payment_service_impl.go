package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type paymentServiceImpl struct {
	repo            repository.PaymentRepository
	repoBill        repository.BillRepository
	repoParticipant repository.ParticipantRepository
	db              *gorm.DB
}

func NewPaymentService(
	repo repository.PaymentRepository,
	repoBill repository.BillRepository,
	repoParticipant repository.ParticipantRepository,
	db *gorm.DB,
) PaymentService {
	return &paymentServiceImpl{
		repo:            repo,
		repoBill:        repoBill,
		repoParticipant: repoParticipant,
		db:              db,
	}
}

// ProcessPayment implements [PaymentService].
func (p *paymentServiceImpl) ProcessPayment(publicID string, req dto.PaymentRq) error {
	bill, err := p.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Split bill not found")
		}
		return model.NewInternalError("Database error")
	}

	access, err := LoadBillAccess(p.db, bill)
	if err != nil {
		return err
	}
	if access.ReadOnly {
		return access.ReadOnlyError()
	}

	var exists bool
	exists, err = p.repoParticipant.ExistsByIDAndBillID(req.ParticipantID, bill.ID)
	if err != nil {
		return model.NewInternalError("Database error")
	}
	if !exists {
		return model.NewBadRequest("participant_id is not a participant of this bill")
	}

	amount, err := money.FromString(req.Amount, bill.Currency)
	if err != nil {
		return model.NewBadRequest(fmt.Sprintf("invalid amount format: %v", err))
	}
	if !amount.IsPositive() {
		return model.NewBadRequest("amount must be greater than zero")
	}

	payment := entity.Payment{
		BillID:    bill.ID,
		ExpenseID: req.ExpenseID,
		PaidBy:    req.ParticipantID,
		Amount:    amount.Amount,
		PaidAt:    time.Now(),
	}

	// Pindah payer: hapus payment lama + upsert yang baru atomik.
	// Tanpa old_participant_id (atau sama dengan yang baru) = upsert biasa.
	if req.OldParticipantID != nil && *req.OldParticipantID != req.ParticipantID {
		oldExists, err := p.repoParticipant.ExistsByIDAndBillID(*req.OldParticipantID, bill.ID)
		if err != nil {
			return model.NewInternalError("Database error")
		}
		if !oldExists {
			return model.NewBadRequest("old_participant_id is not a participant of this bill")
		}

		if err := p.repo.MovePaymentTransaction(bill.ID, req.ExpenseID, *req.OldParticipantID, payment); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("old payment not found")
			}
			return model.NewInternalError("Database error")
		}
		return nil
	}

	return p.repo.SavePaymentTransaction(payment)
}
