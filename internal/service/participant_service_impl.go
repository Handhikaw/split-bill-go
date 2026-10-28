package service

import (
	"errors"

	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type participantServiceImpl struct {
	repo            repository.ParticipantRepository
	repoBill        repository.BillRepository
	repoGoal        repository.GoalRepository
	repoSettlement  repository.SettlementPaymentRepository
	expenseService  ExpenseService
	activityService ActivityService
	db              *gorm.DB
}

func NewParticipantService(
	repo repository.ParticipantRepository,
	repoBill repository.BillRepository,
	expenseService ExpenseService,
	activityService ActivityService,
	repoGoal repository.GoalRepository,
	repoSettlement repository.SettlementPaymentRepository,
	db *gorm.DB,
) ParticipantService {
	return &participantServiceImpl{
		repo:            repo,
		repoBill:        repoBill,
		repoGoal:        repoGoal,
		repoSettlement:  repoSettlement,
		expenseService:  expenseService,
		activityService: activityService,
		db:              db,
	}
}

// validateActor memastikan actor (dari header X-Participant-ID) adalah
// participant bill ini. 0 = tanpa header -> lolos (log di-skip).
// Tanpa validasi ini, id ngarang masuk ke activity_logs -> FK violation.
func (s *participantServiceImpl) validateActor(billID int64, actionParticipant int64) error {
	if actionParticipant == 0 {
		return nil
	}
	ok, err := s.repo.ExistsByIDAndBillID(actionParticipant, billID)
	if err != nil {
		return model.NewInternalError("Database error")
	}
	if !ok {
		return model.NewBadRequest("invalid participant")
	}
	return nil
}

func (s *participantServiceImpl) UpsertParticipant(publicID string, actionParticipant int64, req dto.ParticipantRq) (dto.ParticipantRs, error) {

	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ParticipantRs{}, model.NewNotFound("Public Id not found")
		}
		return dto.ParticipantRs{}, model.NewInternalError("Database error")
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return dto.ParticipantRs{}, err
	}
	if access.ReadOnly {
		return dto.ParticipantRs{}, access.ReadOnlyError()
	}

	if err := s.validateActor(bill.ID, actionParticipant); err != nil {
		return dto.ParticipantRs{}, err
	}

	// Klaim: id terisi = tautkan baris peserta existing ke user,
	// tanpa mengubah nama/status. Return nama existing.
	if req.ID != nil {
		existing, err := s.repo.FindByID(*req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.ParticipantRs{}, model.NewNotFound("Participant not found")
			}
			return dto.ParticipantRs{}, model.NewInternalError("Database error")
		}
		if existing.BillID != bill.ID {
			return dto.ParticipantRs{}, model.NewNotFound("Participant not found in this bill")
		}
		if err := s.repo.SetUserID(existing.ID, req.UserID); err != nil {
			return dto.ParticipantRs{}, model.NewInternalError("Database error")
		}

		s.activityService.Log(entity.ActivityLog{
			BillID:        bill.ID,
			ParticipantID: actionParticipant,
			ActionType:    "CLAIM_PARTICIPANT",
			TargetName:    &existing.Name,
		})

		return dto.ParticipantRs{
			ID:   existing.ID,
			Name: existing.Name,
		}, nil
	}

	isOwner := false
	if req.IsOwner != nil {
		isOwner = *req.IsOwner
	}

	// Baris baru = tambah anggota; klaim (req.ID != nil) tidak kena limit.
	if access.MaxParticipants > 0 {
		participants, err := s.repo.GetParticipants(bill.ID)
		if err != nil {
			return dto.ParticipantRs{}, model.NewInternalError("Database error")
		}
		if len(participants) >= access.MaxParticipants {
			return dto.ParticipantRs{}, access.ParticipantLimitError()
		}
	}

	participant := &entity.Participant{
		BillID:  bill.ID,
		Name:    req.Name,
		UserID:  req.UserID,
		IsOwner: isOwner,
	}

	participantId, err := s.repo.Upsert(participant)
	if err != nil {
		return dto.ParticipantRs{}, model.NewInternalError("Failed Upsert Participant")
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actionParticipant,
		ActionType:    "UPSERT_PARTICIPANT",
		TargetName:    &req.Name,
	})

	return dto.ParticipantRs{
		ID:   participantId,
		Name: req.Name,
	}, nil
}

func (s *participantServiceImpl) RemoveParticipant(publicID string, id int64, actionParticipant int64) error {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Public Id not found")
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

	if err := s.validateActor(bill.ID, actionParticipant); err != nil {
		return err
	}

	// Self-delete (keluar grup): actor ikut terhapus sehingga tidak bisa
	// dipakai sebagai FK activity log -> log tanpa actor.
	logActor := actionParticipant
	if logActor == id {
		logActor = 0
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		participant, err := s.repo.FindByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Participant not found")
			}
			return model.NewInternalError("Database error")
		}
		if participant.BillID != bill.ID {
			return model.NewNotFound("Participant not found in this bill")
		}

		// Ledger patungan tidak boleh hilang diam-diam saat peserta dihapus.
		contributions, err := s.repoGoal.CountContributionsByParticipantID(id)
		if err != nil {
			return model.NewInternalError("Failed read participant contributions")
		}
		if contributions > 0 {
			return model.NewBadRequest("peserta sudah punya kontribusi; hapus kontribusinya dulu")
		}

		settlementsCount, err := s.repoSettlement.CountByParticipantID(id)
		if err != nil {
			return model.NewInternalError("Failed read participant settlements")
		}
		if settlementsCount > 0 {
			return model.NewBadRequest("peserta sudah punya riwayat pelunasan; hapus pelunasannya dulu")
		}

		expenseIDs, err := s.repo.GetAffectedExpenseIDsByParticipantID(tx, id)
		if err != nil {
			return model.NewInternalError("Failed read participant expenses")
		}

		if err := s.repo.DeleteExpenseItemParticipantsByParticipantID(tx, id); err != nil {
			return model.NewInternalError("Failed delete item participants")
		}

		if err := s.repo.DeleteExpenseParticipantsByParticipantID(tx, id); err != nil {
			return model.NewInternalError("Failed delete expense participants")
		}

		if err := s.repo.DeletePaymentsByParticipantID(tx, id); err != nil {
			return model.NewInternalError("Failed delete participant payments")
		}

		for _, expenseID := range expenseIDs {
			if err := s.expenseService.SyncExpenseTotal(tx, expenseID); err != nil {
				return model.NewInternalError("Failed sync expense total")
			}
		}

		if err := tx.Delete(participant).Error; err != nil {
			return model.NewInternalError("Failed delete participant")
		}

		s.activityService.Log(entity.ActivityLog{
			BillID:        participant.BillID,
			ParticipantID: logActor,
			ActionType:    "DELETE_PARTICIPANT",
			TargetName:    &participant.Name,
		})

		return nil
	})
}
