package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type expenseServiceImpl struct {
	repo            repository.ExpenseRepository
	repoBill        repository.BillRepository
	repoParticipant repository.ParticipantRepository
	db              *gorm.DB
	activityService ActivityService
}

func NewExpenseService(repo repository.ExpenseRepository, repoBill repository.BillRepository, repoParticipant repository.ParticipantRepository,
	db *gorm.DB, acactivityService ActivityService) ExpenseService {
	return &expenseServiceImpl{repo: repo, repoBill: repoBill, repoParticipant: repoParticipant, db: db, activityService: acactivityService}
}

func (s *expenseServiceImpl) getBillByPublicID(publicID string) (*entity.Bill, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.NewNotFound("Bill not found")
		}
		return nil, model.NewInternalError("Database error")
	}

	return bill, nil
}

// ensureWritable menolak semua mutasi saat bill read-only (expired/closed free).
func (s *expenseServiceImpl) ensureWritable(bill *entity.Bill) (BillAccess, error) {
	access, err := LoadBillAccess(s.db, bill)
	if err != nil {
		return BillAccess{}, err
	}
	if access.ReadOnly {
		return BillAccess{}, access.ReadOnlyError()
	}
	return access, nil
}

func (s *expenseServiceImpl) validateExpenseBelongsToBill(expense *entity.Expense, billID int64) error {
	if expense.BillID != billID {
		return model.NewNotFound("Expense not found in this bill")
	}

	return nil
}

// resolveActor memastikan actor (dari header X-Participant-ID) adalah
// participant bill ini. 0 = tanpa header -> lolos (log di-skip).
// Id ngarang yang lolos akan menabrak FK activity_logs.
func (s *expenseServiceImpl) resolveActor(billID int64, participantID int64) (int64, error) {
	if participantID == 0 {
		return 0, nil
	}
	ok, err := s.repoParticipant.ExistsByIDAndBillID(participantID, billID)
	if err != nil {
		return 0, model.NewInternalError("Database error")
	}
	if !ok {
		return 0, model.NewBadRequest("invalid participant")
	}
	return participantID, nil
}

// CreateExpense implements [ExpenseService].
func (s *expenseServiceImpl) CreateExpense(req dto.CreateExpenseRq) (dto.CreateExpenseRs, error) {
	bill, err := s.repoBill.GetBillByPublicID(req.PublicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CreateExpenseRs{}, model.NewNotFound("Bill not found")
		}
		return dto.CreateExpenseRs{}, model.NewInternalError("Database error")
	}

	access, err := s.ensureWritable(bill)
	if err != nil {
		return dto.CreateExpenseRs{}, err
	}
	if access.MaxExpenses > 0 {
		expenses, err := s.repo.FindExpenseByBillID(bill.ID)
		if err != nil {
			return dto.CreateExpenseRs{}, model.NewInternalError("Database error")
		}
		if len(expenses) >= access.MaxExpenses {
			return dto.CreateExpenseRs{}, access.ExpenseLimitError()
		}
	}

	var amount money.Money = money.New(0, bill.Currency)
	if req.Amount != nil {
		amount, err = money.FromString(*req.Amount, bill.Currency)
		if err != nil {
			return dto.CreateExpenseRs{}, model.NewBadRequest("invalid amount format")
		}
	}

	var exists bool
	exists, err = s.repoParticipant.ExistsByIDAndBillID(req.CreatedBy, bill.ID)
	if err != nil {
		return dto.CreateExpenseRs{}, model.NewInternalError("Database error")
	}
	if !exists {
		return dto.CreateExpenseRs{}, model.NewBadRequest("created_by is not a participant of this bill")
	}

	expense := entity.Expense{
		BillID:     bill.ID,
		Title:      req.Title,
		Type:       req.Type,
		Amount:     amount.Amount,
		CategoryID: req.CategoryID,
		CreatedBy:  &req.CreatedBy,
	}

	participantShares := make([]dto.ExpenseParticipant, 0, len(req.Participants))

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&expense).Error; err != nil {
			return err
		}

		if len(req.Participants) > 0 {
			shares := amount.SplitRandom(int64(len(req.Participants)))
			for i, participantID := range req.Participants {
				ep := entity.ExpenseParticipant{
					ExpenseID:     expense.ID,
					ParticipantID: participantID,
					ShareAmount:   shares[i].Amount,
					ShareType:     "EQUAL",
				}

				if err := tx.Create(&ep).Error; err != nil {
					return err
				}

				participantShares = append(participantShares, dto.ExpenseParticipant{
					ParticipantID: participantID,
					ShareAmount:   shares[i].String(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return dto.CreateExpenseRs{}, model.NewInternalError("failed to create expense")
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        bill.ID,
		ParticipantID: req.CreatedBy,
		ActionType:    "CREATE_EXPENSE",
		TargetName:    &req.Title,
		Amount:        &amount.Amount,
	})
	//todo code di atas adalah contoh untuk log activity tolong buat untuk CreateExpenseItem, UpdateExpenseItem, DeleteExpenseItem

	return dto.CreateExpenseRs{
		ID:           expense.ID,
		Title:        expense.Title,
		Type:         expense.Type,
		Amount:       amount.String(),
		CategoryID:   expense.CategoryID,
		CreatedBy:    expense.CreatedBy,
		Participants: participantShares,
	}, nil
}

// UpdateExpense implements [ExpenseService].
func (s *expenseServiceImpl) UpdateExpense(publicID string, expenseID int64, participantID int64, req dto.UpdateExpenseRq) (dto.UpdateExpenseRs, error) {
	bill, err := s.getBillByPublicID(publicID)
	if err != nil {
		return dto.UpdateExpenseRs{}, err
	}

	if _, err := s.ensureWritable(bill); err != nil {
		return dto.UpdateExpenseRs{}, err
	}

	actor, err := s.resolveActor(bill.ID, participantID)
	if err != nil {
		return dto.UpdateExpenseRs{}, err
	}

	var updatedExpense *entity.Expense
	var currency string
	participantShares := []dto.ExpenseParticipant{}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var err error

		updatedExpense, err = s.repo.FindExpenseByID(tx, expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense Id not found")
			}
			return model.NewInternalError("Database error")
		}
		if err := s.validateExpenseBelongsToBill(updatedExpense, bill.ID); err != nil {
			return err
		}

		currency, err = s.repo.GetCurrencyByExpenseID(expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense Id not found")
			}
			return model.NewInternalError("Database error")
		}

		isAmountChanged := false
		shouldSyncFromItems := false

		if req.Title != nil {
			updatedExpense.Title = *req.Title
		}
		if req.Amount != nil {
			if updatedExpense.Type == "SIMPLE" {
				amount, err := money.FromString(*req.Amount, currency)
				if err != nil {
					return model.NewBadRequest("invalid amount format")
				}
				if updatedExpense.Amount != amount.Amount {
					isAmountChanged = true
				}
				updatedExpense.Amount = amount.Amount
			} else {
				// Untuk DETAIL/SPLIT_BILL: abaikan amount dari request dan sinkronkan dari item.
				shouldSyncFromItems = true
			}
		}
		if req.Tax != nil {
			tax, err := money.FromString(*req.Tax, currency)
			if err != nil {
				return model.NewBadRequest("invalid tax format")
			}
			updatedExpense.Tax = tax.Amount
		}
		if req.Discount != nil {
			discount, err := money.FromString(*req.Discount, currency)
			if err != nil {
				return model.NewBadRequest("invalid discount format")
			}
			updatedExpense.Discount = discount.Amount
		}
		if req.Charge != nil {
			charge, err := money.FromString(*req.Charge, currency)
			if err != nil {
				return model.NewBadRequest("invalid charge format")
			}
			updatedExpense.Charge = charge.Amount
		}
		if req.CategoryID != nil {
			updatedExpense.CategoryID = req.CategoryID
		}
		if req.CreatedBy != nil {
			exists, err := s.repoParticipant.ExistsByIDAndBillID(*req.CreatedBy, updatedExpense.BillID)
			if err != nil {
				return model.NewInternalError("Database error")
			}
			if !exists {
				return model.NewBadRequest("created_by is not a participant of this bill")
			}
			updatedExpense.CreatedBy = req.CreatedBy
		}

		if err := s.repo.UpdateExpense(tx, updatedExpense); err != nil {
			return model.NewInternalError("failed to update expense")
		}

		if shouldSyncFromItems {
			if err := s.SyncExpenseTotal(tx, expenseID); err != nil {
				return err
			}
			if err := tx.First(updatedExpense, expenseID).Error; err != nil {
				return model.NewInternalError("Database error")
			}
		}

		var targetParticipants []int64
		if req.Participants != nil {
			targetParticipants = req.Participants
		} else if isAmountChanged {
			existingParticipants, err := s.repo.GetExpenseParticipantsByExpenseID(tx, expenseID)
			if err != nil {
				return model.NewInternalError("Database error")
			}
			targetParticipants = make([]int64, 0, len(existingParticipants))
			for _, p := range existingParticipants {
				targetParticipants = append(targetParticipants, p.ParticipantID)
			}
		}

		if targetParticipants != nil {
			for _, participantID := range targetParticipants {
				exists, err := s.repoParticipant.ExistsByIDAndBillID(participantID, updatedExpense.BillID)
				if err != nil {
					return model.NewInternalError("Database error")
				}
				if !exists {
					return model.NewBadRequest("participant is not part of this bill")
				}
			}

			participantEntities := make([]entity.ExpenseParticipant, 0, len(targetParticipants))
			if len(targetParticipants) > 0 {
				shares := money.New(updatedExpense.Amount, currency).SplitRandom(int64(len(targetParticipants)))
				for i, participantID := range targetParticipants {
					participantEntities = append(participantEntities, entity.ExpenseParticipant{
						ExpenseID:     expenseID,
						ParticipantID: participantID,
						ShareAmount:   shares[i].Amount,
						ShareType:     "EQUAL",
					})

					participantShares = append(participantShares, dto.ExpenseParticipant{
						ParticipantID: participantID,
						ShareAmount:   shares[i].String(),
					})
				}
			}

			if err := s.repo.ReplaceExpenseParticipants(tx, expenseID, participantEntities); err != nil {
				return model.NewInternalError("failed to update expense participants")
			}
		} else {
			existingParticipants, err := s.repo.GetExpenseParticipantsByExpenseID(tx, expenseID)
			if err != nil {
				return model.NewInternalError("Database error")
			}

			participantShares = make([]dto.ExpenseParticipant, 0, len(existingParticipants))
			for _, p := range existingParticipants {
				participantShares = append(participantShares, dto.ExpenseParticipant{
					ParticipantID: p.ParticipantID,
					ShareAmount:   money.New(p.ShareAmount, currency).String(),
				})
			}
		}

		return nil
	})
	if err != nil {
		return dto.UpdateExpenseRs{}, err
	}

	s.activityService.Log(entity.ActivityLog{
		BillID:        updatedExpense.BillID,
		ParticipantID: actor,
		ActionType:    "UPDATE_EXPENSE",
		TargetName:    &updatedExpense.Title,
		Amount:        &updatedExpense.Amount,
	})

	return dto.UpdateExpenseRs{
		Title:        updatedExpense.Title,
		Type:         updatedExpense.Type,
		Amount:       money.New(updatedExpense.Amount, currency).String(),
		Tax:          money.New(updatedExpense.Tax, currency).String(),
		Discount:     money.New(updatedExpense.Discount, currency).String(),
		Charge:       money.New(updatedExpense.Charge, currency).String(),
		CategoryID:   updatedExpense.CategoryID,
		CreatedBy:    updatedExpense.CreatedBy,
		Participants: participantShares,
	}, nil
}

// DeleteExpense implements [ExpenseService].
func (s *expenseServiceImpl) DeleteExpense(publicID string, expenseID int64, participantID int64) error {
	bill, err := s.getBillByPublicID(publicID)
	if err != nil {
		return err
	}

	if _, err := s.ensureWritable(bill); err != nil {
		return err
	}

	actor, err := s.resolveActor(bill.ID, participantID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		expense, err := s.repo.FindExpenseByID(tx, expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense not found")
			}
			return model.NewInternalError("Database error")
		}
		if err := s.validateExpenseBelongsToBill(expense, bill.ID); err != nil {
			return err
		}

		if err := s.repo.DeleteItemParticipantsByExpenseID(tx, expenseID); err != nil {
			return model.NewInternalError("failed to delete expense item participants")
		}

		if err := s.repo.DeleteItemsByExpenseID(tx, expenseID); err != nil {
			return model.NewInternalError("failed to delete expense items")
		}

		if err := s.repo.DeleteExpenseParticipantsByExpenseID(tx, expenseID); err != nil {
			return model.NewInternalError("failed to delete expense participants")
		}

		if err := s.repo.DeleteExpense(tx, expense); err != nil {
			return model.NewInternalError("failed to delete expense")
		}

		if err := s.repo.DeletePaymentsByExpenseID(tx, expenseID); err != nil {
			return model.NewInternalError("failed to delete payments")
		}

		fmt.Println(participantID)

		s.activityService.Log(entity.ActivityLog{
			BillID:        expense.BillID,
			ParticipantID: actor,
			ActionType:    "DELETE_EXPENSE",
			TargetName:    &expense.Title,
		})

		return nil
	})
}

// CreateExpenseItem implements [ExpenseService].
func (s *expenseServiceImpl) CreateExpenseItem(publicID string, expenseID int64, participantID int64, req dto.CreateExpenseItemRq) (dto.CreateExpenseItemRs, error) {
	bill, err := s.getBillByPublicID(publicID)
	if err != nil {
		return dto.CreateExpenseItemRs{}, err
	}

	if _, err := s.ensureWritable(bill); err != nil {
		return dto.CreateExpenseItemRs{}, err
	}

	actor, err := s.resolveActor(bill.ID, participantID)
	if err != nil {
		return dto.CreateExpenseItemRs{}, err
	}

	// Item boleh tanpa peserta (belum assign): share dibuat saat
	// peserta diisi menyusul via PATCH. Tanpa guard ini SplitRandom(0) panic.
	if len(req.Participants) == 0 {
		currency, err := s.repo.GetCurrencyByExpenseID(expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.CreateExpenseItemRs{}, model.NewNotFound("Expense Id not found")
			}
			return dto.CreateExpenseItemRs{}, model.NewInternalError("Database error")
		}

		price, err := money.FromString(req.Price, currency)
		if err != nil {
			return dto.CreateExpenseItemRs{}, err
		}

		var item entity.ExpenseItem
		err = s.db.Transaction(func(tx *gorm.DB) error {
			expense, err := s.repo.FindExpenseByID(tx, expenseID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return model.NewNotFound("Expense not found")
				}
				return model.NewInternalError("Database error")
			}
			if err := s.validateExpenseBelongsToBill(expense, bill.ID); err != nil {
				return err
			}

			item = entity.ExpenseItem{
				ExpenseID: expenseID,
				Name:      req.Name,
				Quantity:  req.Quantity,
				Price:     price.Amount,
			}
			if err := s.repo.CreateItem(tx, &item); err != nil {
				return err
			}
			if err := s.SyncExpenseTotal(tx, expenseID); err != nil {
				return err
			}

			s.activityService.Log(entity.ActivityLog{
				BillID:        expense.BillID,
				ParticipantID: actor,
				ActionType:    "CREATE_EXPENSE_ITEM",
				TargetName:    &item.Name,
				Amount:        &price.Amount,
			})

			return nil
		})
		if err != nil {
			return dto.CreateExpenseItemRs{}, err
		}

		return dto.CreateExpenseItemRs{
			ID:           item.ID,
			Name:         item.Name,
			Price:        price.String(),
			Quantity:     item.Quantity,
			Total:        price.Multiply(int64(req.Quantity)).String(),
			Participants: []int64{},
		}, nil
	}

	currency, err := s.repo.GetCurrencyByExpenseID(expenseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CreateExpenseItemRs{}, model.NewNotFound("Expense Id not found")
		}
		return dto.CreateExpenseItemRs{}, model.NewInternalError("Database error")
	}

	price, err := money.FromString(req.Price, currency)
	if err != nil {
		return dto.CreateExpenseItemRs{}, err
	}

	// Deklarasi variabel di luar agar bisa diakses untuk response
	var item entity.ExpenseItem
	var totalPrice money.Money

	err = s.db.Transaction(func(tx *gorm.DB) error {
		expense, err := s.repo.FindExpenseByID(tx, expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense not found")
			}
			return model.NewInternalError("Database error")
		}
		if err := s.validateExpenseBelongsToBill(expense, bill.ID); err != nil {
			return err
		}

		for _, pID := range req.Participants {
			exists, err := s.repoParticipant.ExistsByIDAndBillID(pID, expense.BillID)
			if err != nil {
				return model.NewInternalError("Database error")
			}
			if !exists {
				return model.NewBadRequest("participant is not part of this bill")
			}
		}

		// Insert Item
		item = entity.ExpenseItem{
			ExpenseID: expenseID,
			Name:      req.Name,
			Quantity:  req.Quantity,
			Price:     price.Amount,
		}

		if err := s.repo.CreateItem(tx, &item); err != nil {
			return err // Return error saja di dalam transaction
		}

		totalPrice = price.Multiply(int64(req.Quantity))
		countParticipants := int64(len(req.Participants))
		shares := totalPrice.SplitRandom(countParticipants)

		// Insert Participant
		if len(req.Participants) > 0 {

			for i, pID := range req.Participants {

				ep := entity.ExpenseParticipant{
					ExpenseID:     expenseID,
					ParticipantID: pID,
					ShareAmount:   shares[i].Amount,
					ShareType:     "EQUAL",
				}

				if err := s.repoParticipant.UpsertParticipantShare(tx, &ep); err != nil {
					return err
				}
			}
		}

		// Insert Item Participant
		if len(req.Participants) > 0 {

			for i, piID := range req.Participants {

				eip := entity.ExpenseItemParticipant{
					ItemID:        item.ID,
					ParticipantID: piID,
					ShareAmount:   shares[i].Amount,
				}
				if err := s.repo.CreateItemParticipant(tx, &eip); err != nil {
					return err
				}

			}
		}

		if err := s.SyncExpenseTotal(tx, expenseID); err != nil {
			return err
		}

		s.activityService.Log(entity.ActivityLog{
			BillID:        expense.BillID,
			ParticipantID: actor,
			ActionType:    "CREATE_EXPENSE_ITEM",
			TargetName:    &item.Name,
			Amount:        &totalPrice.Amount,
		})

		return nil
	})

	if err != nil {
		return dto.CreateExpenseItemRs{}, err
	}

	res := dto.CreateExpenseItemRs{
		ID:           item.ID,
		Name:         item.Name,
		Price:        price.String(),
		Quantity:     item.Quantity,
		Total:        totalPrice.String(),
		Participants: req.Participants,
	}

	return res, nil
}

// UpdateExpenseItem implements [ExpenseService].
func (s *expenseServiceImpl) UpdateExpenseItem(publicID string, expenseID int64, itemID int64, participantID int64, req dto.UpdateExpenseItemRq) error {
	bill, err := s.getBillByPublicID(publicID)
	if err != nil {
		return err
	}

	if _, err := s.ensureWritable(bill); err != nil {
		return err
	}

	actor, err := s.resolveActor(bill.ID, participantID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		currency, err := s.repo.GetCurrencyByExpenseID(expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense Id not found")
			}
			return model.NewInternalError("Database error")
		}

		expense, err := s.repo.FindExpenseByID(tx, expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense not found")
			}
			return model.NewInternalError("Database error")
		}
		if err := s.validateExpenseBelongsToBill(expense, bill.ID); err != nil {
			return err
		}

		// 1. Ambil data lama
		var item entity.ExpenseItem
		if err := tx.First(&item, itemID).Error; err != nil {
			return err
		}
		if item.ExpenseID != expenseID {
			return model.NewNotFound("Item not found in this expense")
		}

		isPriceChanged := false

		// 2. Update field secara parsial
		if req.Name != nil {
			item.Name = *req.Name

			s.activityService.Log(entity.ActivityLog{
				BillID:        expense.BillID,
				ParticipantID: actor,
				ActionType:    "UPDATE_EXPENSE_ITEM_NAME",
				TargetName:    &item.Name,
				Amount:        nil,
			})
		}
		if req.Price != nil {
			p, _ := money.FromString(*req.Price, currency)
			if item.Price != p.Amount {
				item.Price = p.Amount
				isPriceChanged = true

				s.activityService.Log(entity.ActivityLog{
					BillID:        expense.BillID,
					ParticipantID: actor,
					ActionType:    "UPDATE_EXPENSE_ITEM_PRICE",
					TargetName:    &item.Name,
					Amount:        &p.Amount,
				})
			}

		}
		if req.Quantity != nil {
			if item.Quantity != *req.Quantity {
				item.Quantity = *req.Quantity
				isPriceChanged = true

				s.activityService.Log(entity.ActivityLog{
					BillID:        expense.BillID,
					ParticipantID: actor,
					ActionType:    "UPDATE_EXPENSE_ITEM_QTY",
					TargetName:    &item.Name,
					Amount:        nil,
				})
			}
		}

		// 3. Update item
		if err := s.repo.UpdateItem(tx, &item); err != nil {
			return err
		}

		// 4. Logic Recalculation (Hanya jika harga/qty berubah ATAU partisipan dikirim)
		if req.Participants != nil || isPriceChanged {
			// Jika req.Participants nil tapi harga berubah, ambil partisipan lama dari DB
			targetParticipants := req.Participants
			if targetParticipants == nil {
				targetParticipants, err = s.repo.GetParticipantIDsByItemID(tx, itemID)
				if err != nil {
					return err
				}
			}

			for _, pID := range targetParticipants {
				exists, err := s.repoParticipant.ExistsByIDAndBillID(pID, expense.BillID)
				if err != nil {
					return model.NewInternalError("Database error")
				}
				if !exists {
					return model.NewBadRequest("participant is not part of this bill")
				}
			}

			// 5. Delete detail lama
			if err := s.repo.DeleteItemParticipant(tx, itemID); err != nil {
				return err
			}

			if len(targetParticipants) > 0 {
				totalPrice := money.New(item.Price, currency).Multiply(int64(item.Quantity))
				shares := totalPrice.SplitRandom(int64(len(targetParticipants)))

				for i, pID := range targetParticipants {
					eip := entity.ExpenseItemParticipant{
						ItemID:        itemID,
						ParticipantID: pID,
						ShareAmount:   shares[i].Amount,
					}
					if err := s.repo.CreateItemParticipant(tx, &eip); err != nil {
						return err
					}
				}
			}

		}

		return s.SyncExpenseTotal(tx, expenseID)
	})
}

// DeleteExpenseItem implements [ExpenseService].
func (s *expenseServiceImpl) DeleteExpenseItem(publicID string, expenseID int64, itemID int64, participantID int64) error {
	bill, err := s.getBillByPublicID(publicID)
	if err != nil {
		return err
	}

	if _, err := s.ensureWritable(bill); err != nil {
		return err
	}

	actor, err := s.resolveActor(bill.ID, participantID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {

		expense, err := s.repo.FindExpenseByID(tx, expenseID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Expense Id not found")
			}
			return model.NewInternalError("Database error")
		}
		if err := s.validateExpenseBelongsToBill(expense, bill.ID); err != nil {
			return err
		}

		// 1. Cari item dulu untuk mendapatkan ExpenseID-nya
		item, err := s.repo.FindItemByID(tx, itemID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.NewNotFound("Item not found")
			}
			return err
		}
		if item.ExpenseID != expenseID {
			return model.NewNotFound("Item not found in this expense")
		}

		// 2. Hapus detail partisipan di tabel junction (ExpenseItemParticipant)
		// Ini harus dilakukan sebelum hapus item-nya agar tidak ada data 'yatim'
		if err := s.repo.DeleteItemParticipant(tx, itemID); err != nil {
			return err
		}

		// 3. Hapus data Item
		if err := s.repo.DeleteItem(tx, item); err != nil {
			return err
		}

		// 4. SINKRONISASI TOTAL!
		if err := s.SyncExpenseTotal(tx, expenseID); err != nil {
			return err
		}

		s.activityService.Log(entity.ActivityLog{
			BillID:        expense.BillID,
			ParticipantID: actor,
			ActionType:    "DELETE_EXPENSE_ITEM",
			TargetName:    &item.Name,
			Amount:        nil,
		})

		return nil
	})
}

// GetCategory implements [ExpenseService].
func (s *expenseServiceImpl) GetCategory(userID *int64) ([]dto.CategoryRs, error) {
	var categories []entity.Category

	query := s.db.Model(&entity.Category{}).
		Where("is_default = ?", true)

	if userID != nil {
		query = query.Or("user_id = ?", *userID)
	}

	if err := query.Order("id ASC").Find(&categories).Error; err != nil {
		return nil, model.NewInternalError("Database error")
	}

	result := make([]dto.CategoryRs, 0, len(categories))
	for _, category := range categories {
		result = append(result, dto.CategoryRs{
			ID:        category.ID,
			Name:      category.Name,
			Icon:      category.Icon,
			ColorCode: category.ColorCode,
			IsDefault: category.IsDefault,
		})
	}

	return result, nil
}

// UpsertCategory implements [ExpenseService].
func (s *expenseServiceImpl) UpsertCategory(req dto.CategoryRq) error {
	if req.UserID <= 0 {
		return model.NewBadRequest("invalid user_id")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.NewBadRequest("name is required")
	}

	icon := ""
	if req.Icon != nil {
		icon = *req.Icon
	}

	colorCode := ""
	if req.ColorCode != nil {
		colorCode = *req.ColorCode
	}

	if req.ID == nil {
		var existingCount int64
		if err := s.db.Model(&entity.Category{}).
			Where("user_id = ? AND name = ?", req.UserID, name).
			Count(&existingCount).Error; err != nil {
			return model.NewInternalError("Database error")
		}
		if existingCount > 0 {
			return model.NewBadRequest("category already exists")
		}

		userID := req.UserID
		newCategory := entity.Category{
			Name:      name,
			Icon:      icon,
			ColorCode: colorCode,
			IsDefault: false,
			UserID:    &userID,
		}

		if err := s.db.Create(&newCategory).Error; err != nil {
			return model.NewInternalError("Database error")
		}

		return nil
	}

	var category entity.Category
	if err := s.db.Where("id = ? AND user_id = ?", *req.ID, req.UserID).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.NewNotFound("category not found")
		}
		return model.NewInternalError("Database error")
	}

	var existingCount int64
	if err := s.db.Model(&entity.Category{}).
		Where("user_id = ? AND name = ? AND id <> ?", req.UserID, name, *req.ID).
		Count(&existingCount).Error; err != nil {
		return model.NewInternalError("Database error")
	}
	if existingCount > 0 {
		return model.NewBadRequest("category already exists")
	}

	category.Name = name
	if req.Icon != nil {
		category.Icon = *req.Icon
	}
	if req.ColorCode != nil {
		category.ColorCode = *req.ColorCode
	}

	if err := s.db.Save(&category).Error; err != nil {
		return model.NewInternalError("Database error")
	}

	return nil
}

// SyncExpenseTotal implements [ExpenseService].
func (s *expenseServiceImpl) SyncExpenseTotal(tx *gorm.DB, expenseID int64) error {
	var expense entity.Expense
	if err := tx.Select("id", "type").First(&expense, expenseID).Error; err != nil {
		return err
	}

	// Expense SIMPLE tidak disinkronkan dari item.
	if expense.Type == "SIMPLE" {
		return nil
	}

	// 1. Rebalance dulu detail share di expense_item_participant per item.
	// Ini penting saat jumlah participant item berubah (mis. setelah delete participant).
	type ExpenseItemRow struct {
		ID       int64
		Price    int64
		Quantity int
	}
	var items []ExpenseItemRow

	err := tx.Table("expense_item").
		Select("id, price, quantity").
		Where("expense_id = ?", expenseID).
		Scan(&items).Error
	if err != nil {
		return err
	}

	var totalExpenseAmount int64
	for _, item := range items {
		total := item.Price * int64(item.Quantity)
		totalExpenseAmount += total

		var participantIDs []int64
		err := tx.Table("expense_item_participant").
			Where("item_id = ?", item.ID).
			Order("participant_id ASC").
			Pluck("participant_id", &participantIDs).Error
		if err != nil {
			return err
		}

		if len(participantIDs) == 0 {
			continue
		}

		base := total / int64(len(participantIDs))
		remainder := total % int64(len(participantIDs))

		for i, participantID := range participantIDs {
			share := base
			if int64(i) < remainder {
				share++
			}

			if err := tx.Table("expense_item_participant").
				Where("item_id = ? AND participant_id = ?", item.ID, participantID).
				Update("share_amount", share).Error; err != nil {
				return err
			}
		}
	}

	// 2. Sinkronkan amount expense dari subtotal semua item.
	if err := tx.Model(&entity.Expense{}).
		Where("id = ?", expenseID).
		Update("amount", totalExpenseAmount).Error; err != nil {
		return err
	}

	// 3. Reset semua share_amount partisipan di Expense ini ke 0.
	// Penting: Agar orang yang sudah tidak punya item lagi, tagihannya jadi 0.
	err = tx.Model(&entity.ExpenseParticipant{}).
		Where("expense_id = ?", expenseID).
		Update("share_amount", 0).Error
	if err != nil {
		return err
	}

	// 4. Ambil akumulasi SUM dari detail per item
	// Kita Join ke expense_item karena filter-nya adalah expense_id
	type Result struct {
		ParticipantID int64
		TotalShare    int64
	}
	var results []Result

	err = tx.Table("expense_item_participant").
		Select("expense_item_participant.participant_id, SUM(expense_item_participant.share_amount) as total_share").
		Joins("JOIN expense_item ON expense_item.id = expense_item_participant.item_id").
		Where("expense_item.expense_id = ?", expenseID).
		Group("expense_item_participant.participant_id").
		Scan(&results).Error

	if err != nil {
		return err
	}

	// 5. Update kembali ke tabel ExpenseParticipant (Summary table)
	for _, res := range results {
		ep := entity.ExpenseParticipant{
			ExpenseID:     expenseID,
			ParticipantID: res.ParticipantID,
			ShareAmount:   res.TotalShare,
			ShareType:     "EQUAL",
		}

		// Gunakan Upsert agar kalau record belum ada (partisipan baru), dia akan Create
		if err := s.repoParticipant.UpsertParticipantShare(tx, &ep); err != nil {
			return err
		}
	}

	return nil
}
