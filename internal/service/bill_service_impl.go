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
	activityService ActivityService
	db              *gorm.DB
}

func NewBillService(repo repository.BillRepository,
	repoExpense repository.ExpenseRepository,
	repoParticipant repository.ParticipantRepository,
	repoPayment repository.PaymentRepository,
	activityService ActivityService,
	db *gorm.DB) BillService {
	return &billServiceImpl{repo: repo,
		repoExpense:     repoExpense,
		repoParticipant: repoParticipant,
		repoPayment:     repoPayment,
		activityService: activityService,
		db:              db}
}

func (s *billServiceImpl) lookupUser(userID *int64) (entity.User, error) {
	var user entity.User
	if userID == nil {
		return user, nil
	}
	if err := s.db.Select("id, role, username").Where("id = ?", *userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, model.NewNotFound("user not found")
		}
		return entity.User{}, model.NewInternalError("Database error")
	}
	return user, nil
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
func (s *billServiceImpl) GetSplitBill(publicID string) (*dto.GetSplitBillRs, error) {
	// 1. Ambil data mentah dari Repository
	bill, err := s.repo.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.GetSplitBillRs{}, model.NewNotFound("Split bill not found")
		}
		return &dto.GetSplitBillRs{}, model.NewInternalError("Database error")
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

	// 2. Jalankan Engine yang kita buat sebelumnya
	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, payments, bill.Currency)
	if err != nil {
		return &dto.GetSplitBillRs{}, model.NewInternalError("Failed calculate balances")
	}
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

	// 3. Mapping ke Response DTO
	return &dto.GetSplitBillRs{
		ID:            bill.ID,
		ExpenseID:     expenseID,
		Status:        bill.Status,
		Name:          bill.Name,
		Currency:      bill.Currency,
		TotalAmount:   money.New(s.CalculateTotalAmount(expenses, items), bill.Currency).String(),
		Tax:           money.New(exp.Tax, bill.Currency).String(),
		ServiceCharge: money.New(exp.Charge, bill.Currency).String(),
		Discount:      money.New(exp.Discount, bill.Currency).String(),
		Items:         itemsDTO,
		Participants:  balanceDTO,
		Settlements:   settlements,
	}, nil
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
func (s *billServiceImpl) GetGroupBill(publicID string) (dto.GetGroupBillRs, error) {
	bill, err := s.repo.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.GetGroupBillRs{}, model.NewNotFound("Group bill not found")
		}
		return dto.GetGroupBillRs{}, model.NewInternalError("Database error")
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

	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, payments, bill.Currency)
	if err != nil {
		return dto.GetGroupBillRs{}, model.NewInternalError("Failed calculate balances")
	}

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

	return dto.GetGroupBillRs{
		ID:           bill.ID,
		Status:       bill.Status,
		Name:         bill.Name,
		Currency:     bill.Currency,
		TotalAmount:  money.New(s.CalculateTotalAmount(expenses, items), bill.Currency).String(),
		Expense:      expenseDTO,
		Participants: balanceDTO,
		Settlements:  settlements,
	}, nil
}
