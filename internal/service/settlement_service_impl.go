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

type settlementServiceImpl struct {
	repoSettlement  repository.SettlementPaymentRepository
	repoBill        repository.BillRepository
	repoParticipant repository.ParticipantRepository
	activityService ActivityService
	db              *gorm.DB
}

func NewSettlementService(
	repoSettlement repository.SettlementPaymentRepository,
	repoBill repository.BillRepository,
	repoParticipant repository.ParticipantRepository,
	activityService ActivityService,
	db *gorm.DB,
) SettlementService {
	return &settlementServiceImpl{
		repoSettlement:  repoSettlement,
		repoBill:        repoBill,
		repoParticipant: repoParticipant,
		activityService: activityService,
		db:              db,
	}
}

func (s *settlementServiceImpl) loadActor(
	bill *entity.Bill,
	actorParticipantID int64,
	access BillAccess,
) (*entity.Participant, error) {
	if actorParticipantID <= 0 {
		return nil, model.NewBadRequest("klaim sebagai peserta dulu")
	}
	actor, err := s.repoParticipant.FindByID(actorParticipantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.NewBadRequest("klaim sebagai peserta dulu")
		}
		return nil, model.NewInternalError("Database error")
	}
	if actor.BillID != bill.ID {
		return nil, model.NewBadRequest("invalid participant")
	}

	// Hybrid: bill Premium ATAU actor Premium.
	if !access.Premium {
		premium, err := IsPremiumUser(s.db, actor.UserID)
		if err != nil {
			return nil, err
		}
		if !premium {
			return nil, model.NewBadRequest("fitur mark-as-paid khusus Premium; upgrade ke Pro")
		}
	}

	return actor, nil
}

func (s *settlementServiceImpl) targetName(
	fromID, toID int64,
) string {
	fromName, toName := "?", "?"
	if p, err := s.repoParticipant.FindByID(fromID); err == nil && p != nil {
		fromName = p.Name
	}
	if p, err := s.repoParticipant.FindByID(toID); err == nil && p != nil {
		toName = p.Name
	}
	return fmt.Sprintf("%s → %s", fromName, toName)
}

func (s *settlementServiceImpl) Create(
	publicID string,
	actorParticipantID int64,
	req dto.SettlementPaymentRq,
) (dto.SettlementPaymentItem, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.SettlementPaymentItem{}, model.NewNotFound("Bill not found")
		}
		return dto.SettlementPaymentItem{}, model.NewInternalError("Database error")
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return dto.SettlementPaymentItem{}, err
	}
	if access.ReadOnly {
		return dto.SettlementPaymentItem{}, access.ReadOnlyError()
	}

	actor, err := s.loadActor(bill, actorParticipantID, access)
	if err != nil {
		return dto.SettlementPaymentItem{}, err
	}

	if req.FromParticipantID == req.ToParticipantID {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("from dan to harus berbeda")
	}

	exists, err := s.repoParticipant.ExistsByIDAndBillID(req.FromParticipantID, bill.ID)
	if err != nil {
		return dto.SettlementPaymentItem{}, model.NewInternalError("Database error")
	}
	if !exists {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("from_participant_id bukan peserta bill ini")
	}
	exists, err = s.repoParticipant.ExistsByIDAndBillID(req.ToParticipantID, bill.ID)
	if err != nil {
		return dto.SettlementPaymentItem{}, model.NewInternalError("Database error")
	}
	if !exists {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("to_participant_id bukan peserta bill ini")
	}

	if actor.ID != req.FromParticipantID && actor.ID != req.ToParticipantID && !actor.IsOwner {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("hanya pihak terkait atau owner yang bisa mencatat pelunasan")
	}

	amount, err := money.FromString(req.Amount, bill.Currency)
	if err != nil {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("invalid amount format")
	}
	if !amount.IsPositive() {
		return dto.SettlementPaymentItem{}, model.NewBadRequest("amount must be greater than zero")
	}

	payment := &entity.SettlementPayment{
		BillID:            bill.ID,
		FromParticipantID: req.FromParticipantID,
		ToParticipantID:   req.ToParticipantID,
		Amount:            amount.Amount,
		Note:              cleanOptionalText(req.Note, 255),
		PaidAt:            time.Now(),
	}
	if err := s.repoSettlement.Create(payment); err != nil {
		return dto.SettlementPaymentItem{}, model.NewInternalError("failed to save settlement payment")
	}

	target := s.targetName(payment.FromParticipantID, payment.ToParticipantID)
	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actor.ID,
		ActionType:    "ADD_SETTLEMENT_PAYMENT",
		TargetName:    &target,
		Amount:        &payment.Amount,
	})

	fromName, toName := "?", "?"
	if p, err := s.repoParticipant.FindByID(payment.FromParticipantID); err == nil && p != nil {
		fromName = p.Name
	}
	if p, err := s.repoParticipant.FindByID(payment.ToParticipantID); err == nil && p != nil {
		toName = p.Name
	}

	return dto.SettlementPaymentItem{
		ID:     payment.ID,
		From:   dto.PersonInfo{ID: payment.FromParticipantID, Name: fromName},
		To:     dto.PersonInfo{ID: payment.ToParticipantID, Name: toName},
		Amount: money.New(payment.Amount, bill.Currency).String(),
		Note:   payment.Note,
		PaidAt: payment.PaidAt,
	}, nil
}

func (s *settlementServiceImpl) Delete(
	publicID string,
	settlementID int64,
	actorParticipantID int64,
) error {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Bill not found")
		}
		return model.NewInternalError("Database error")
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return err
	}
	if access.ReadOnly {
		return access.ReadOnlyError()
	}

	if actorParticipantID <= 0 {
		return model.NewBadRequest("klaim sebagai peserta dulu")
	}
	actor, err := s.repoParticipant.FindByID(actorParticipantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewBadRequest("klaim sebagai peserta dulu")
		}
		return model.NewInternalError("Database error")
	}
	if actor.BillID != bill.ID {
		return model.NewBadRequest("invalid participant")
	}
	if err := requireOwnerParticipant(s.repoParticipant, bill.ID, actor.ID); err != nil {
		return err
	}

	row, err := s.repoSettlement.FindByIDAndBillID(settlementID, bill.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Pelunasan tidak ditemukan")
		}
		return model.NewInternalError("Database error")
	}

	if err := s.repoSettlement.Delete(row); err != nil {
		return model.NewInternalError("failed to delete settlement payment")
	}

	target := s.targetName(row.FromParticipantID, row.ToParticipantID)
	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actor.ID,
		ActionType:    "DELETE_SETTLEMENT_PAYMENT",
		TargetName:    &target,
		Amount:        &row.Amount,
	})

	return nil
}
