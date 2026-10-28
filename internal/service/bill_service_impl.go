package service

import (
	"errors"
	"strings"
	"time"

	"github.com/harkandavez/split-bill-go/internal/helper/engine"
	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type billServiceImpl struct {
	repo            repository.BillRepository
	repoExpense     repository.ExpenseRepository
	repoParticipant repository.ParticipantRepository
	repoPayment     repository.PaymentRepository
	repoGoal        repository.GoalRepository
	repoSettlement  repository.SettlementPaymentRepository
	activityService ActivityService
	db              *gorm.DB
}

func NewBillService(repo repository.BillRepository,
	repoExpense repository.ExpenseRepository,
	repoParticipant repository.ParticipantRepository,
	repoPayment repository.PaymentRepository,
	repoGoal repository.GoalRepository,
	repoSettlement repository.SettlementPaymentRepository,
	activityService ActivityService,
	db *gorm.DB) BillService {
	return &billServiceImpl{repo: repo,
		repoExpense:     repoExpense,
		repoParticipant: repoParticipant,
		repoPayment:     repoPayment,
		repoGoal:        repoGoal,
		repoSettlement:  repoSettlement,
		activityService: activityService,
		db:              db}
}

func (s *billServiceImpl) lookupUser(userID *int64) (entity.User, error) {
	return lookupUserByID(s.db, userID)
}

func resolveOwnerName(creatorName *string, username string) string {
	if creatorName != nil && strings.TrimSpace(*creatorName) != "" {
		name := strings.TrimSpace(*creatorName)
		if len(name) > 100 {
			return name[:100]
		}
		return name
	}
	if strings.TrimSpace(username) != "" {
		return strings.TrimSpace(username)
	}
	return "Owner"
}

func (s *billServiceImpl) CreateSplitBill(req dto.CreateSpiltBillRq, actorUserID *int64) (dto.CreateSpiltBillRs, error) {

	if !money.IsSupportedCurrency(req.Currency) {
		return dto.CreateSpiltBillRs{}, model.NewBadRequest("unsupported currency")
	}

	// Identitas murni dari JWT (OptionalAuthMiddleware). Nil = guest.
	effectiveID := actorUserID
	user, err := s.lookupUser(effectiveID)
	if err != nil {
		return dto.CreateSpiltBillRs{}, err
	}

	publicId, err := gonanoid.New(20)
	if err != nil {
		return dto.CreateSpiltBillRs{}, model.NewInternalError("failed to generate public id")
	}

	group := &entity.Bill{
		PublicID:  publicId,
		Name:      req.Title,
		Currency:  req.Currency,
		Kind:      "split",
		Status:    "ACTIVE",
		CreatedBy: effectiveID,
	}

	expense := &entity.Expense{
		Title: req.Title,
		Type:  "SPLIT_BILL",
	}

	owner := &entity.Participant{
		Name:    resolveOwnerName(req.CreatorName, user.Username),
		UserID:  effectiveID,
		IsOwner: true,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		expense.BillID = group.ID
		if err := tx.Create(expense).Error; err != nil {
			return err
		}
		owner.BillID = group.ID
		return tx.Create(owner).Error
	}); err != nil {
		return dto.CreateSpiltBillRs{}, model.NewInternalError("failed to process transaction: " + err.Error())
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        group.ID,
		ParticipantID: owner.ID,
		ActionType:    "CREATE_SPLIT_BILL",
		TargetName:    &group.Name,
	})

	return dto.CreateSpiltBillRs{
		PublicId:           group.PublicID,
		OwnerParticipantID: owner.ID,
	}, nil
}

// GetSplitBil implements [BillService].
func (s *billServiceImpl) GetSplitBill(publicID string, viewerUserID *int64) (*dto.GetSplitBillRs, error) {
	// 1. Ambil data mentah dari Repository
	bill, err := s.repo.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.GetSplitBillRs{}, model.NewNotFound("Split bill not found")
		}
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}
	if bill.Kind == "goal" {
		return &dto.GetSplitBillRs{}, model.NewNotFound("Split bill not found")
	}

	participants, err := s.repoParticipant.GetParticipants(bill.ID)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}
	expenses, err := s.repoExpense.FindExpenseByBillID(bill.ID)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}

	// Splitbill = 1 Expense
	if len(expenses) == 0 {
		return &dto.GetSplitBillRs{}, model.NewNotFound("Split bill expense not found")
	}
	exp := expenses[0]
	var expenseID int64 = exp.ID

	items, err := s.repoExpense.GetItemsByExpenseID(expenseID)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}
	itemIDs := make([]int64, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}
	var itemParts []entity.ExpenseItemParticipant
	if len(itemIDs) > 0 {
		itemParts, err = s.repoExpense.GetItemParticipantsByItemIDs(itemIDs)
		if err != nil {
			return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
		}
	}
	expenseParts, err := s.repoExpense.GetExpenseParticipantsByExpenseIDs([]int64{expenseID})
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}
	payments, err := s.repoPayment.GetPaymentsByBillID(bill.ID)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}
	settlementRows, err := s.repoSettlement.ListByBillID(bill.ID)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
	}

	// 2. Jalankan Engine yang kita buat sebelumnya
	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, payments, bill.Currency)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Failed calculate balances")
	}
	balances = engine.ApplySettlementPayments(balances, settlementRows)
	settlements, err := engine.GenerateSettlements(balances)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Failed generate settlements")
	}

	balanceDTO := make([]dto.ParticipantBalance, 0, len(balances))
	for _, b := range balances {
		balanceDTO = append(balanceDTO, engine.ToParticipantBalanceDTO(b))
	}

	itemsDTO := s.MapItemsDTO(items, itemParts, participants, bill.Currency)
	if itemsDTO == nil {
		itemsDTO = []dto.ExpenseItem{}
	}
	if settlements == nil {
		settlements = []dto.Settlement{}
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return &dto.GetSplitBillRs{}, err
	}
	viewerPremium, err := IsPremiumUser(s.db, viewerUserID)
	if err != nil {
		return &dto.GetSplitBillRs{}, err
	}

	// 3. Mapping ke Response DTO
	return &dto.GetSplitBillRs{
		ID:                 bill.ID,
		ExpenseID:          expenseID,
		Status:             bill.Status,
		Name:               bill.Name,
		Currency:           bill.Currency,
		TotalAmount:        money.New(s.CalculateTotalAmount(expenses, items), bill.Currency).String(),
		Tax:                money.New(exp.Tax, bill.Currency).String(),
		ServiceCharge:      money.New(exp.Charge, bill.Currency).String(),
		Discount:           money.New(exp.Discount, bill.Currency).String(),
		Plan:               access.Plan,
		ReadOnly:           access.ReadOnly,
		ViewerIsPremium:    viewerPremium,
		Items:              itemsDTO,
		Participants:       balanceDTO,
		Settlements:        settlements,
		SettlementPayments: s.mapSettlementPayments(settlementRows, participants, bill.Currency),
	}, nil
}

func (s *billServiceImpl) mapSettlementPayments(
	rows []entity.SettlementPayment,
	participants []entity.Participant,
	currency string,
) []dto.SettlementPaymentItem {
	nameByID := make(map[int64]string, len(participants))
	for _, p := range participants {
		nameByID[p.ID] = p.Name
	}
	nameOf := func(id int64) string {
		if n := nameByID[id]; n != "" {
			return n
		}
		return "Mantan anggota"
	}

	out := make([]dto.SettlementPaymentItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.SettlementPaymentItem{
			ID:     row.ID,
			From:   dto.PersonInfo{ID: row.FromParticipantID, Name: nameOf(row.FromParticipantID)},
			To:     dto.PersonInfo{ID: row.ToParticipantID, Name: nameOf(row.ToParticipantID)},
			Amount: money.New(row.Amount, currency).String(),
			Note:   row.Note,
			PaidAt: row.PaidAt,
		})
	}
	return out
}

func (s *billServiceImpl) CalculateTotalAmount(expenses []entity.Expense, items []entity.ExpenseItem) int64 {
	itemTotalsByExpense := make(map[int64]int64)
	itemCountByExpense := make(map[int64]int)

	for _, item := range items {
		itemTotalsByExpense[item.ExpenseID] += item.Price * int64(item.Quantity)
		itemCountByExpense[item.ExpenseID]++
	}

	var total int64
	for _, exp := range expenses {
		baseAmount := exp.Amount
		if itemCountByExpense[exp.ID] > 0 {
			baseAmount = itemTotalsByExpense[exp.ID]
		}

		total += baseAmount + exp.Tax + exp.Charge - exp.Discount
	}

	return total
}

func (s *billServiceImpl) MapItemsDTO(
	items []entity.ExpenseItem,
	itemParts []entity.ExpenseItemParticipant,
	participants []entity.Participant,
	currency string,
) []dto.ExpenseItem {

	// 1. Buat Map Nama Partisipan untuk lookup cepat
	// key: participant_id, value: name
	participantMap := make(map[int64]string)
	for _, p := range participants {
		participantMap[p.ID] = p.Name
	}

	// 2. Kelompokkan ItemParticipants berdasarkan ItemID
	// key: item_id, value: list of participants who ate this
	partsByItem := make(map[int64][]dto.ExpenseItemParticipant)
	for _, ip := range itemParts {
		name := participantMap[ip.ParticipantID]

		dto := dto.ExpenseItemParticipant{
			ParticipantID: ip.ParticipantID,
			Name:          name,
			ShareAmount:   money.New(ip.ShareAmount, currency).String(),
		}
		partsByItem[ip.ItemID] = append(partsByItem[ip.ItemID], dto)
	}

	// 3. Gabungkan ke dalam slice ItemDTO
	result := make([]dto.ExpenseItem, 0, len(items))
	for _, item := range items {
		subtotal := item.Price * int64(item.Quantity)

		itemDTO := dto.ExpenseItem{
			ID:           item.ID,
			Name:         item.Name,
			Price:        money.New(item.Price, currency).String(),
			Quantity:     item.Quantity,
			Subtotal:     money.New(subtotal, currency).String(),
			Participants: partsByItem[item.ID], // ambil dari map yang dibuat di step 2
		}

		// Pastikan kalau kosong tidak nil di JSON, tapi empty array []
		if itemDTO.Participants == nil {
			itemDTO.Participants = []dto.ExpenseItemParticipant{}
		}

		result = append(result, itemDTO)
	}

	return result
}

func (s *billServiceImpl) JoinSplitBill(publicID string, req dto.JoinSplitBillRq) error {
	// Cari group berdasarkan public_id
	bill, err := s.repo.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("Split bill not found")
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
	if access.MaxParticipants > 0 {
		participants, err := s.repoParticipant.GetParticipants(bill.ID)
		if err != nil {
			return model.NewInternalError("Database error")
		}
		if len(participants) >= access.MaxParticipants {
			return access.ParticipantLimitError()
		}
	}

	// Buat entity Participant
	participant := &entity.Participant{
		BillID:  bill.ID,
		Name:    req.Name,
		IsOwner: false, // Karena ini join via link
		UserID:  req.UserID,
	}

	// Simpan
	if err := s.repo.CreateParticipant(participant); err != nil {
		return model.NewInternalError("Failed to join group")
	}

	return nil
}

// CreateGroupBill implements [BillService].
func (s *billServiceImpl) CreateGroupBill(req dto.CreateGroupBillRq, actorUserID *int64) (dto.CreateGroupBillRs, error) {

	if !money.IsSupportedCurrency(req.Currency) {
		return dto.CreateGroupBillRs{}, model.NewBadRequest("unsupported currency")
	}

	// Identitas murni dari JWT (OptionalAuthMiddleware). Nil = guest.
	effectiveID := actorUserID
	user, err := s.lookupUser(effectiveID)
	if err != nil {
		return dto.CreateGroupBillRs{}, err
	}

	publicID, err := gonanoid.New(20)
	if err != nil {
		return dto.CreateGroupBillRs{}, model.NewInternalError("failed to generate public id")
	}

	group := &entity.Bill{
		PublicID:  publicID,
		Name:      req.Title,
		Currency:  req.Currency,
		Kind:      "group",
		Status:    "ACTIVE",
		CreatedBy: effectiveID,
	}

	isPremium := strings.EqualFold(user.Role, "PREMIUM")
	if !isPremium {
		t := time.Now().Add(48 * time.Hour)
		group.ExpiredAt = &t
	} else {
		group.ExpiredAt = nil
	}

	owner := &entity.Participant{
		Name:    resolveOwnerName(req.CreatorName, user.Username),
		UserID:  effectiveID,
		IsOwner: true,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		owner.BillID = group.ID
		return tx.Create(owner).Error
	}); err != nil {
		return dto.CreateGroupBillRs{}, model.NewInternalError("failed to create group bill: " + err.Error())
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        group.ID,
		ParticipantID: owner.ID,
		ActionType:    "CREATE_GROUP_BILL",
		TargetName:    &group.Name,
	})

	return dto.CreateGroupBillRs{
		PublicId:           group.PublicID,
		Title:              group.Name,
		Status:             group.Status,
		ExpiredAt:          group.ExpiredAt,
		OwnerParticipantID: owner.ID,
	}, nil
}

// GetGroupBill implements [BillService].
func (s *billServiceImpl) GetGroupBill(publicID string, viewerUserID *int64) (dto.GetGroupBillRs, error) {
	bill, err := s.repo.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGroupBillRs{}, model.NewNotFound("Group bill not found")
		}
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
	}
	if bill.Kind == "goal" {
		return dto.GetGroupBillRs{}, model.NewNotFound("Group bill not found")
	}

	participants, err := s.repoParticipant.GetParticipants(bill.ID)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
	}

	expenses, err := s.repoExpense.FindExpenseByBillID(bill.ID)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
	}

	expenseIDs := make([]int64, 0, len(expenses))
	for _, exp := range expenses {
		expenseIDs = append(expenseIDs, exp.ID)
	}

	items := []entity.ExpenseItem{}
	if len(expenseIDs) > 0 {
		items, err = s.repoExpense.GetItemsByExpenseIDs(expenseIDs)
		if err != nil {
			return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
		}
	}

	itemIDs := make([]int64, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}

	itemParts := []entity.ExpenseItemParticipant{}
	if len(itemIDs) > 0 {
		itemParts, err = s.repoExpense.GetItemParticipantsByItemIDs(itemIDs)
		if err != nil {
			return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
		}
	}

	expenseParts := []entity.ExpenseParticipant{}
	if len(expenseIDs) > 0 {
		expenseParts, err = s.repoExpense.GetExpenseParticipantsByExpenseIDs(expenseIDs)
		if err != nil {
			return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
		}
	}

	payments, err := s.repoPayment.GetPaymentsByBillID(bill.ID)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
	}
	settlementRows, err := s.repoSettlement.ListByBillID(bill.ID)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
	}

	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, payments, bill.Currency)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Failed calculate balances")
	}
	balances = engine.ApplySettlementPayments(balances, settlementRows)

	settlements, err := engine.GenerateSettlements(balances)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Failed generate settlements")
	}
	if settlements == nil {
		settlements = []dto.Settlement{}
	}

	balanceDTO := make([]dto.ParticipantBalance, 0, len(balances))
	for _, b := range balances {
		balanceDTO = append(balanceDTO, engine.ToParticipantBalanceDTO(b))
	}

	participantNameByID := make(map[int64]string, len(participants))
	for _, p := range participants {
		participantNameByID[p.ID] = p.Name
	}

	categoryNameByID := map[int64]string{}
	categoryIDs := make([]int64, 0, len(expenses))
	seenCategory := map[int64]bool{}
	for _, exp := range expenses {
		if exp.CategoryID != nil && !seenCategory[*exp.CategoryID] {
			seenCategory[*exp.CategoryID] = true
			categoryIDs = append(categoryIDs, *exp.CategoryID)
		}
	}
	if len(categoryIDs) > 0 {
		type categoryRow struct {
			ID   int64
			Name string
		}
		var rows []categoryRow
		if err := s.db.Table("category").Select("id, name").Where("id IN ?", categoryIDs).Scan(&rows).Error; err != nil {
			return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
		}
		for _, row := range rows {
			categoryNameByID[row.ID] = row.Name
		}
	}

	expensePartsByExpenseID := make(map[int64][]dto.ExpenseParticipant, len(expenses))
	for _, part := range expenseParts {
		expensePartsByExpenseID[part.ExpenseID] = append(expensePartsByExpenseID[part.ExpenseID], dto.ExpenseParticipant{
			ParticipantID: part.ParticipantID,
			ShareAmount:   money.New(part.ShareAmount, bill.Currency).String(),
		})
	}

	expenseDTO := make([]dto.Expense, 0, len(expenses))
	for _, exp := range expenses {
		expenseItems := make([]entity.ExpenseItem, 0)
		for _, item := range items {
			if item.ExpenseID == exp.ID {
				expenseItems = append(expenseItems, item)
			}
		}

		itemsDTO := s.MapItemsDTO(expenseItems, itemParts, participants, bill.Currency)

		createdByName := ""
		if exp.CreatedBy != nil {
			createdByName = participantNameByID[*exp.CreatedBy]
		}

		categoryName := ""
		if exp.CategoryID != nil {
			categoryName = categoryNameByID[*exp.CategoryID]
		}

		expenseParticipants := expensePartsByExpenseID[exp.ID]
		if expenseParticipants == nil {
			expenseParticipants = []dto.ExpenseParticipant{}
		}

		expenseDTO = append(expenseDTO, dto.Expense{
			ID:           exp.ID,
			Title:        exp.Title,
			Amount:       money.New(exp.Amount, bill.Currency).String(),
			Tax:          money.New(exp.Tax, bill.Currency).String(),
			Discount:     money.New(exp.Discount, bill.Currency).String(),
			Charge:       money.New(exp.Charge, bill.Currency).String(),
			Category:     categoryName,
			Created_By:   createdByName,
			Created_At:   exp.CreatedAt,
			Participants: expenseParticipants,
			Items:        &itemsDTO,
		})
	}

	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return dto.GetGroupBillRs{}, err
	}
	viewerPremium, err := IsPremiumUser(s.db, viewerUserID)
	if err != nil {
		return dto.GetGroupBillRs{}, err
	}

	return dto.GetGroupBillRs{
		ID:              bill.ID,
		Status:          bill.Status,
		Name:            bill.Name,
		Currency:        bill.Currency,
		TotalAmount:     money.New(s.CalculateTotalAmount(expenses, items), bill.Currency).String(),
		Plan:            access.Plan,
		ReadOnly:        access.ReadOnly,
		ExpiredAt:       bill.ExpiredAt,
		ViewerIsPremium: viewerPremium,
		Limits: dto.BillLimits{
			MaxParticipants: access.MaxParticipants,
			MaxExpenses:     access.MaxExpenses,
		},
		Usage: dto.BillUsage{
			ParticipantCount: len(participants),
			ExpenseCount:     len(expenses),
		},
		Expense:            expenseDTO,
		Participants:       balanceDTO,
		Settlements:        settlements,
		SettlementPayments: s.mapSettlementPayments(settlementRows, participants, bill.Currency),
	}, nil
}

// ListMyBills implements [BillService].
// "Bill saya" = saya pembuatnya (created_by) ATAU saya pesertanya
// (participant.user_id). Terakhir aktif dulu (activity terakhir,
// fallback created_at). Limit <= 0 = default 20.
func (s *billServiceImpl) ListMyBills(userID int64, limit int) ([]dto.MyBillItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	var bills []entity.Bill
	err := s.db.
		Where("created_by = ? OR id IN (?)",
			userID,
			s.db.Table("participant").Select("bill_id").Where("user_id = ?", userID),
		).
		Order(gorm.Expr("COALESCE((SELECT MAX(created_at) FROM activity_logs WHERE bill_id = bill.id), created_at) DESC")).
		Limit(limit).
		Find(&bills).Error
	if err != nil {
		return nil, model.NewInternalError("Database error")
	}

	items := make([]dto.MyBillItem, 0, len(bills))
	for _, bill := range bills {
		summary, err := s.summarizeBillForUser(bill, userID)
		if err != nil {
			return nil, err
		}
		items = append(items, dto.MyBillItem{
			PublicID:        bill.PublicID,
			Kind:            summary.kind,
			Title:           bill.Name,
			Currency:        bill.Currency,
			TotalAmount:     summary.total,
			Status:          bill.Status,
			MyNet:           summary.myNet,
			MemberCount:     summary.memberCount,
			TargetAmount:    summary.targetAmount,
			CollectedAmount: summary.collectedAmount,
			UpdatedAt:       summary.updated,
		})
	}
	return items, nil
}

type myBillSummary struct {
	kind            string
	total           string
	targetAmount    *string
	collectedAmount *string
	myNet           *string
	memberCount     int
	updated         time.Time
}

func (s *billServiceImpl) summarizeBillForUser(
	bill entity.Bill, userID int64,
) (myBillSummary, error) {
	summary := myBillSummary{kind: bill.Kind, updated: bill.CreatedAt}
	if summary.kind == "" {
		summary.kind = "group"
	}

	participants, err := s.repoParticipant.GetParticipants(bill.ID)
	if err != nil {
		return myBillSummary{}, model.NewInternalError("Database error")
	}
	summary.memberCount = len(participants)

	var lastAct *time.Time
	if err := s.db.Table("activity_logs").
		Select("MAX(created_at)").Where("bill_id = ?", bill.ID).
		Scan(&lastAct).Error; err == nil && lastAct != nil {
		summary.updated = *lastAct
	}

	expenses, err := s.repoExpense.FindExpenseByBillID(bill.ID)
	if err != nil {
		return myBillSummary{}, model.NewInternalError("Database error")
	}

	// Data lama (sebelum kolom kind) tetap dikenali sebagai split.
	if summary.kind == "group" {
		for _, exp := range expenses {
			if exp.Type == "SPLIT_BILL" {
				summary.kind = "split"
				break
			}
		}
	}

	// Patungan tidak punya expense: total = target, net = null (bukan utang).
	if summary.kind == "goal" {
		goal, err := s.repoGoal.GetGoalByBillID(bill.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myBillSummary{}, model.NewNotFound("goal bill not found")
			}
			return myBillSummary{}, model.NewInternalError("Database error")
		}
		collected, err := s.repoGoal.SumContributions(bill.ID)
		if err != nil {
			return myBillSummary{}, model.NewInternalError("Database error")
		}
		target := money.New(goal.TargetAmount, bill.Currency).String()
		collectedStr := money.New(collected, bill.Currency).String()
		summary.total = target
		summary.targetAmount = &target
		summary.collectedAmount = &collectedStr
		return summary, nil
	}

	expenseIDs := make([]int64, 0, len(expenses))
	for _, exp := range expenses {
		expenseIDs = append(expenseIDs, exp.ID)
	}

	items := []entity.ExpenseItem{}
	itemParts := []entity.ExpenseItemParticipant{}
	if len(expenseIDs) > 0 {
		items, err = s.repoExpense.GetItemsByExpenseIDs(expenseIDs)
		if err != nil {
			return myBillSummary{}, model.NewInternalError("Database error")
		}
		itemIDs := make([]int64, 0, len(items))
		for _, item := range items {
			itemIDs = append(itemIDs, item.ID)
		}
		if len(itemIDs) > 0 {
			itemParts, err = s.repoExpense.GetItemParticipantsByItemIDs(itemIDs)
			if err != nil {
				return myBillSummary{}, model.NewInternalError("Database error")
			}
		}
	}

	payments, err := s.repoPayment.GetPaymentsByBillID(bill.ID)
	if err != nil {
		return myBillSummary{}, model.NewInternalError("Database error")
	}
	settlementRows, err := s.repoSettlement.ListByBillID(bill.ID)
	if err != nil {
		return myBillSummary{}, model.NewInternalError("Database error")
	}

	var expenseParts []entity.ExpenseParticipant
	if len(expenseIDs) > 0 {
		expenseParts, err = s.repoExpense.GetExpenseParticipantsByExpenseIDs(expenseIDs)
		if err != nil {
			return myBillSummary{}, model.NewInternalError("Database error")
		}
	}

	summary.total = money.New(s.CalculateTotalAmount(expenses, items), bill.Currency).String()

	// Net saya = gabungan semua baris peserta milik user ini.
	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, payments, bill.Currency)
	if err != nil {
		return myBillSummary{}, model.NewInternalError("Failed calculate balances")
	}
	balances = engine.ApplySettlementPayments(balances, settlementRows)
	var net *money.Money
	for _, b := range balances {
		p := participantByID(participants, b.ParticipantID)
		if p == nil || p.UserID == nil || *p.UserID != userID {
			continue
		}
		sum := b.NetBalance
		if net == nil {
			n := sum
			net = &n
		} else {
			n := net.Add(sum)
			net = &n
		}
	}
	if net != nil {
		str := net.String()
		summary.myNet = &str
	}

	return summary, nil
}

func participantByID(participants []entity.Participant, id int64) *entity.Participant {
	for i := range participants {
		if participants[i].ID == id {
			return &participants[i]
		}
	}
	return nil
}
