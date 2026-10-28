package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type goalServiceImpl struct {
	repoGoal        repository.GoalRepository
	repoBill        repository.BillRepository
	repoParticipant repository.ParticipantRepository
	activityService ActivityService
	db              *gorm.DB
}

func NewGoalService(
	repoGoal repository.GoalRepository,
	repoBill repository.BillRepository,
	repoParticipant repository.ParticipantRepository,
	activityService ActivityService,
	db *gorm.DB,
) GoalService {
	return &goalServiceImpl{
		repoGoal:        repoGoal,
		repoBill:        repoBill,
		repoParticipant: repoParticipant,
		activityService: activityService,
		db:              db,
	}
}

func cleanOptionalText(v *string, max int) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	if len(s) > max {
		s = s[:max]
	}
	return &s
}

func billAccessOrError(db *gorm.DB, bill *entity.Bill) (BillAccess, error) {
	access, err := LoadBillAccess(db, bill)
	if err != nil {
		return BillAccess{}, err
	}
	if access.ReadOnly {
		return BillAccess{}, access.ReadOnlyError()
	}
	return access, nil
}

func (s *goalServiceImpl) requireOwner(billID, actorParticipantID int64) error {
	if actorParticipantID <= 0 {
		return model.NewUnauthorized("klaim sebagai owner dulu")
	}
	p, err := s.repoParticipant.FindByID(actorParticipantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewUnauthorized("hanya owner yang bisa mengubah patungan")
		}
		return model.NewInternalError("Database error")
	}
	if p.BillID != billID || !p.IsOwner {
		return model.NewUnauthorized("hanya owner yang bisa mengubah patungan")
	}
	return nil
}

func (s *goalServiceImpl) CreateGoalBill(req dto.CreateGoalBillRq, actorUserID *int64) (dto.CreateGoalBillRs, error) {
	if !money.IsSupportedCurrency(req.Currency) {
		return dto.CreateGoalBillRs{}, model.NewBadRequest("unsupported currency")
	}

	target, err := money.FromString(req.TargetAmount, req.Currency)
	if err != nil {
		return dto.CreateGoalBillRs{}, model.NewBadRequest("invalid target_amount format")
	}
	if !target.IsPositive() {
		return dto.CreateGoalBillRs{}, model.NewBadRequest("target_amount must be greater than zero")
	}

	user, err := lookupUserByID(s.db, actorUserID)
	if err != nil {
		return dto.CreateGoalBillRs{}, err
	}

	publicID, err := gonanoid.New(20)
	if err != nil {
		return dto.CreateGoalBillRs{}, model.NewInternalError("failed to generate public id")
	}

	bill := &entity.Bill{
		PublicID:  publicID,
		Name:      req.Title,
		Currency:  req.Currency,
		Kind:      "goal",
		Status:    "ACTIVE",
		CreatedBy: actorUserID,
	}
	// Free: data bertahan 30 hari. Premium: permanen.
	if !strings.EqualFold(user.Role, "PREMIUM") {
		t := time.Now().Add(30 * 24 * time.Hour)
		bill.ExpiredAt = &t
	}

	goal := &entity.GoalBill{
		TargetAmount: target.Amount,
		Description:  cleanOptionalText(req.Description, 255),
	}
	owner := &entity.Participant{
		Name:    resolveOwnerName(req.CreatorName, user.Username),
		UserID:  actorUserID,
		IsOwner: true,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bill).Error; err != nil {
			return err
		}
		goal.BillID = bill.ID
		if err := tx.Create(goal).Error; err != nil {
			return err
		}
		owner.BillID = bill.ID
		return tx.Create(owner).Error
	}); err != nil {
		return dto.CreateGoalBillRs{}, model.NewInternalError("failed to create goal bill: " + err.Error())
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: owner.ID,
		ActionType:    "CREATE_GOAL_BILL",
		TargetName:    &bill.Name,
		Amount:        &goal.TargetAmount,
	})

	return dto.CreateGoalBillRs{
		PublicID:           bill.PublicID,
		Title:              bill.Name,
		Status:             bill.Status,
		TargetAmount:       money.New(goal.TargetAmount, bill.Currency).String(),
		ExpiredAt:          bill.ExpiredAt,
		OwnerParticipantID: owner.ID,
	}, nil
}

func (s *goalServiceImpl) GetGoalBill(publicID string) (dto.GetGoalBillRs, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGoalBillRs{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}

	goal, err := s.repoGoal.GetGoalByBillID(bill.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGoalBillRs{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}

	participants, err := s.repoParticipant.GetParticipants(bill.ID)
	if err != nil {
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].ID < participants[j].ID
	})

	rows, err := s.repoGoal.ListContributions(bill.ID)
	if err != nil {
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}

	nameByID := make(map[int64]string, len(participants))
	for _, p := range participants {
		nameByID[p.ID] = p.Name
	}

	type total struct {
		amount int64
		count  int
	}
	totals := make(map[int64]*total, len(participants))
	var collected int64

	contributions := make([]dto.GoalContributionItem, 0, len(rows))
	for _, c := range rows {
		collected += c.Amount
		name := nameByID[c.ParticipantID]
		if name == "" {
			name = "Mantan anggota"
		}
		if totals[c.ParticipantID] == nil {
			totals[c.ParticipantID] = &total{}
		}
		totals[c.ParticipantID].amount += c.Amount
		totals[c.ParticipantID].count++

		contributions = append(contributions, dto.GoalContributionItem{
			ID:            c.ID,
			ParticipantID: c.ParticipantID,
			Name:          name,
			Amount:        money.New(c.Amount, bill.Currency).String(),
			Note:          c.Note,
			PaidAt:        c.PaidAt,
		})
	}

	progress := make([]dto.GoalParticipantProgress, 0, len(participants))
	for _, p := range participants {
		t := totals[p.ID]
		var amount int64
		var count int
		if t != nil {
			amount = t.amount
			count = t.count
		}
		progress = append(progress, dto.GoalParticipantProgress{
			ParticipantID:     p.ID,
			Name:              p.Name,
			IsOwner:           p.IsOwner,
			TotalContributed:  money.New(amount, bill.Currency).String(),
			ContributionCount: count,
		})
	}

	remaining := goal.TargetAmount - collected
	if remaining < 0 {
		remaining = 0
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return dto.GetGoalBillRs{}, err
	}

	return dto.GetGoalBillRs{
		ID:             bill.ID,
		Status:         bill.Status,
		Name:           bill.Name,
		Currency:       bill.Currency,
		TargetAmount:   money.New(goal.TargetAmount, bill.Currency).String(),
		TotalCollected: money.New(collected, bill.Currency).String(),
		Remaining:      money.New(remaining, bill.Currency).String(),
		Description:    goal.Description,
		ExpiredAt:      bill.ExpiredAt,
		Plan:           access.Plan,
		ReadOnly:       access.ReadOnly,
		Limits: dto.BillLimits{
			MaxParticipants: access.MaxParticipants,
			MaxExpenses:     access.MaxExpenses,
		},
		Usage: dto.BillUsage{
			ParticipantCount: len(participants),
			ExpenseCount:     0,
		},
		Participants:  progress,
		Contributions: contributions,
	}, nil
}

// resolveContributionActor memilih aktor log yang valid (header), fallback ke
// peserta yang tercatat di kontribusi agar FK activity_logs tidak pecah.
func (s *goalServiceImpl) resolveContributionActor(billID, actorID, fallbackID int64) int64 {
	if actorID > 0 {
		ok, err := s.repoParticipant.ExistsByIDAndBillID(actorID, billID)
		if err == nil && ok {
			return actorID
		}
	}
	ok, err := s.repoParticipant.ExistsByIDAndBillID(fallbackID, billID)
	if err == nil && ok {
		return fallbackID
	}
	return 0
}

func (s *goalServiceImpl) AddContribution(publicID string, actorParticipantID int64, req dto.GoalContributionRq) (dto.GoalContributionItem, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GoalContributionItem{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GoalContributionItem{}, model.NewInternalError("Database error")
	}

	if _, err := s.repoGoal.GetGoalByBillID(bill.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GoalContributionItem{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GoalContributionItem{}, model.NewInternalError("Database error")
	}

	if _, err := billAccessOrError(s.db, bill); err != nil {
		return dto.GoalContributionItem{}, err
	}

	exists, err := s.repoParticipant.ExistsByIDAndBillID(req.ParticipantID, bill.ID)
	if err != nil {
		return dto.GoalContributionItem{}, model.NewInternalError("Database error")
	}
	if !exists {
		return dto.GoalContributionItem{}, model.NewBadRequest("participant_id is not a participant of this bill")
	}

	amount, err := money.FromString(req.Amount, bill.Currency)
	if err != nil {
		return dto.GoalContributionItem{}, model.NewBadRequest("invalid amount format")
	}
	if !amount.IsPositive() {
		return dto.GoalContributionItem{}, model.NewBadRequest("amount must be greater than zero")
	}

	row := &entity.Contribution{
		BillID:        bill.ID,
		ParticipantID: req.ParticipantID,
		Amount:        amount.Amount,
		Note:          cleanOptionalText(req.Note, 255),
		PaidAt:        time.Now(),
	}
	if err := s.repoGoal.CreateContribution(row); err != nil {
		return dto.GoalContributionItem{}, model.NewInternalError("failed to save contribution")
	}

	name := ""
	if p, err := s.repoParticipant.FindByID(row.ParticipantID); err == nil && p != nil {
		name = p.Name
	}

	actor := s.resolveContributionActor(bill.ID, actorParticipantID, row.ParticipantID)
	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actor,
		ActionType:    "ADD_CONTRIBUTION",
		TargetName:    &name,
		Amount:        &row.Amount,
	})

	return dto.GoalContributionItem{
		ID:            row.ID,
		ParticipantID: row.ParticipantID,
		Name:          name,
		Amount:        money.New(row.Amount, bill.Currency).String(),
		Note:          row.Note,
		PaidAt:        row.PaidAt,
	}, nil
}

func (s *goalServiceImpl) DeleteContribution(publicID string, contributionID int64, actorParticipantID int64) error {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Patungan tidak ditemukan")
		}
		return model.NewInternalError("Database error")
	}

	row, err := s.repoGoal.FindContributionByIDAndBillID(contributionID, bill.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Kontribusi tidak ditemukan")
		}
		return model.NewInternalError("Database error")
	}

	if _, err := billAccessOrError(s.db, bill); err != nil {
		return err
	}

	if err := s.repoGoal.DeleteContribution(row); err != nil {
		return model.NewInternalError("failed to delete contribution")
	}

	name := ""
	if p, err := s.repoParticipant.FindByID(row.ParticipantID); err == nil && p != nil {
		name = p.Name
	}

	actor := s.resolveContributionActor(bill.ID, actorParticipantID, row.ParticipantID)
	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actor,
		ActionType:    "DELETE_CONTRIBUTION",
		TargetName:    &name,
		Amount:        &row.Amount,
	})

	return nil
}

func (s *goalServiceImpl) UpdateGoalBill(
	publicID string,
	actorParticipantID int64,
	req dto.UpdateGoalBillRq,
) (dto.GetGoalBillRs, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGoalBillRs{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}

	if _, err := s.repoGoal.GetGoalByBillID(bill.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGoalBillRs{}, model.NewNotFound("Patungan tidak ditemukan")
		}
		return dto.GetGoalBillRs{}, model.NewInternalError("Database error")
	}

	if _, err := billAccessOrError(s.db, bill); err != nil {
		return dto.GetGoalBillRs{}, err
	}
	if err := s.requireOwner(bill.ID, actorParticipantID); err != nil {
		return dto.GetGoalBillRs{}, err
	}

	billFields := map[string]any{}
	goalFields := map[string]any{}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return dto.GetGoalBillRs{}, model.NewBadRequest("title tidak boleh kosong")
		}
		if len(title) > 150 {
			title = title[:150]
		}
		billFields["name"] = title
	}

	if req.TargetAmount != nil {
		target, err := money.FromString(*req.TargetAmount, bill.Currency)
		if err != nil {
			return dto.GetGoalBillRs{}, model.NewBadRequest("invalid target_amount format")
		}
		if !target.IsPositive() {
			return dto.GetGoalBillRs{}, model.NewBadRequest("target_amount must be greater than zero")
		}
		goalFields["target_amount"] = target.Amount
	}

	if req.Description != nil {
		if desc := cleanOptionalText(req.Description, 255); desc != nil {
			goalFields["description"] = *desc
		} else {
			goalFields["description"] = nil
		}
	}

	if len(billFields) == 0 && len(goalFields) == 0 {
		return dto.GetGoalBillRs{}, model.NewBadRequest("tidak ada field yang diubah")
	}

	if err := s.repoGoal.UpdateGoalBill(bill.ID, billFields, goalFields); err != nil {
		return dto.GetGoalBillRs{}, model.NewInternalError("failed to update goal bill")
	}

	logName := bill.Name
	if v, ok := billFields["name"].(string); ok {
		logName = v
	}
	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: actorParticipantID,
		ActionType:    "UPDATE_GOAL_BILL",
		TargetName:    &logName,
	})

	return s.GetGoalBill(publicID)
}
