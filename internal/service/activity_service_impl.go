package service

import (
	"errors"
	"fmt"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type activityServiceImpl struct {
	repo            repository.ActivityRepository
	repoBill        repository.BillRepository
	repoParticipant repository.ParticipantRepository
}

func NewActivityService(
	repo repository.ActivityRepository,
	repoBill repository.BillRepository,
	repoParticipant repository.ParticipantRepository,
) ActivityService {
	return &activityServiceImpl{repo: repo, repoBill: repoBill, repoParticipant: repoParticipant}
}

// Log implements [ActivityService].
func (a *activityServiceImpl) Log(log entity.ActivityLog) {
	// Jika tidak ada participant ID, mungkin tidak perlu dicatat log-nya
	// atau dicatat sebagai "System/Guest"
	if log.ParticipantID == 0 {
		return
	}

	// Jalankan secara ASYNC menggunakan Goroutine
	go func(l entity.ActivityLog) {
		err := a.repo.Save(l)
		if err != nil {
			// Karena async, kita hanya bisa log internal jika gagal
			fmt.Printf("Error saving activity log: %v\n", err)
		}
	}(log)
}

// List implements [ActivityService].
func (a *activityServiceImpl) List(publicID string, limit int) ([]dto.ActivityItem, error) {
	bill, err := a.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.NewNotFound("Bill not found")
		}
		return nil, model.NewInternalError("Database error")
	}

	logs, err := a.repo.ListByBillID(bill.ID, limit)
	if err != nil {
		return nil, model.NewInternalError("Database error")
	}

	participants, err := a.repoParticipant.GetParticipants(bill.ID)
	if err != nil {
		return nil, model.NewInternalError("Database error")
	}
	nameByID := make(map[int64]string, len(participants))
	for _, p := range participants {
		nameByID[p.ID] = p.Name
	}

	items := make([]dto.ActivityItem, 0, len(logs))
	for _, l := range logs {
		actor, ok := nameByID[l.ParticipantID]
		if !ok {
			// Aktor sudah dihapus dari bill — tetap tampil jujur.
			actor = "Mantan anggota"
		}
		var amount *string
		if l.Amount != nil {
			s := money.New(*l.Amount, bill.Currency).String()
			amount = &s
		}
		items = append(items, dto.ActivityItem{
			ID:         l.ID,
			ActorName:  actor,
			ActionType: l.ActionType,
			TargetName: l.TargetName,
			Amount:     amount,
			CreatedAt:  l.CreatedAt,
		})
	}
	return items, nil
}
