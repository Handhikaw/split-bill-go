package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type expenseRepositoryImpl struct {
	db *gorm.DB
}

// GetItemsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetItemsByExpenseID(expenseId int64) ([]entity.ExpenseItem, error) {
	var item []entity.ExpenseItem
	err := e.db.Where("expense_id = ?", expenseId).Find(&item).Error
	return item, err
}

// GetItemParticipantsByItemIDs implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetItemParticipantsByItemIDs(itemIds []int64) ([]entity.ExpenseItemParticipant, error) {
	var itemParts []entity.ExpenseItemParticipant
	err := e.db.Where("item_id IN ?", itemIds).Find(&itemParts).Error
	return itemParts, err
}

// GetItemsByExpenseIDs implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetItemsByExpenseIDs(expenseIds []int64) ([]entity.ExpenseItem, error) {
	var items []entity.ExpenseItem
	// Menggunakan IN query: SELECT * FROM expense_items WHERE expense_id IN (1, 2, 3)
	err := e.db.Where("expense_id IN ?", expenseIds).Find(&items).Error
	return items, err
}

// GetExpenseParticipantsByExpenseIDs implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetExpenseParticipantsByExpenseIDs(expenseIds []int64) ([]entity.ExpenseParticipant, error) {
	var parts []entity.ExpenseParticipant
	err := e.db.Where("expense_id IN ?", expenseIds).Find(&parts).Error
	return parts, err
}

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &expenseRepositoryImpl{db: db}
}

func (e *expenseRepositoryImpl) GetCurrencyByExpenseID(ID int64) (string, error) {
	var currency string
	err := e.db.Table("bill b").
		Select("currency").
		Joins("join expense e on e.bill_id = b.id").
		Where("e.id = ?", ID).
		Scan(&currency).Error
	if err != nil {
		return "", err
	}
	if currency == "" {
		return "", gorm.ErrRecordNotFound
	}
	return currency, nil
}

// FindExpenseByBillID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) FindExpenseByBillID(billID int64) ([]entity.Expense, error) {
	var expenses []entity.Expense
	err := e.db.Where("bill_id = ?", billID).Find(&expenses).Error
	return expenses, err
}

func (e *expenseRepositoryImpl) CreateItem(tx *gorm.DB, item *entity.ExpenseItem) error {
	return tx.Create(item).Error
}

// FindExpenseByID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) FindExpenseByID(tx *gorm.DB, expenseID int64) (*entity.Expense, error) {
	var expense entity.Expense
	if err := tx.First(&expense, expenseID).Error; err != nil {
		return nil, err
	}
	return &expense, nil
}

// UpdateExpense implements [ExpenseRepository].
func (e *expenseRepositoryImpl) UpdateExpense(tx *gorm.DB, expense *entity.Expense) error {
	return tx.Save(expense).Error
}

// DeleteExpense implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteExpense(tx *gorm.DB, expense *entity.Expense) error {
	return tx.Delete(expense).Error
}

// ReplaceExpenseParticipants implements [ExpenseRepository].
func (e *expenseRepositoryImpl) ReplaceExpenseParticipants(tx *gorm.DB, expenseID int64, participants []entity.ExpenseParticipant) error {
	if err := tx.Where("expense_id = ?", expenseID).Delete(&entity.ExpenseParticipant{}).Error; err != nil {
		return err
	}

	if len(participants) == 0 {
		return nil
	}

	return tx.Create(&participants).Error
}

// GetExpenseParticipantsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetExpenseParticipantsByExpenseID(tx *gorm.DB, expenseID int64) ([]entity.ExpenseParticipant, error) {
	var participants []entity.ExpenseParticipant
	err := tx.Where("expense_id = ?", expenseID).Find(&participants).Error
	return participants, err
}

// DeleteItemParticipantsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteItemParticipantsByExpenseID(tx *gorm.DB, expenseID int64) error {
	return tx.Where("item_id IN (?)", tx.Model(&entity.ExpenseItem{}).Select("id").Where("expense_id = ?", expenseID)).
		Delete(&entity.ExpenseItemParticipant{}).Error
}

// DeleteItemsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteItemsByExpenseID(tx *gorm.DB, expenseID int64) error {
	return tx.Where("expense_id = ?", expenseID).Delete(&entity.ExpenseItem{}).Error
}

// DeleteExpenseParticipantsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteExpenseParticipantsByExpenseID(tx *gorm.DB, expenseID int64) error {
	return tx.Where("expense_id = ?", expenseID).Delete(&entity.ExpenseParticipant{}).Error
}

// DeletePaymentsByExpenseID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeletePaymentsByExpenseID(tx *gorm.DB, expenseID int64) error {
	return tx.Where("expense_id = ?", expenseID).Delete(&entity.Payment{}).Error
}

// FindItemByID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) FindItemByID(tx *gorm.DB, itemID int64) (*entity.ExpenseItem, error) {
	var item entity.ExpenseItem
	if err := tx.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem implements [ExpenseRepository].
func (e *expenseRepositoryImpl) UpdateItem(tx *gorm.DB, item *entity.ExpenseItem) error {
	return tx.Save(&item).Error
}

// DeleteItem implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteItem(tx *gorm.DB, item *entity.ExpenseItem) error {
	return tx.Delete(&item).Error
}

func (e *expenseRepositoryImpl) CreateItemParticipant(tx *gorm.DB, itemParticipant *entity.ExpenseItemParticipant) error {
	return tx.Create(itemParticipant).Error
}

// GetParticipantIDsByItemID implements [ExpenseRepository].
func (e *expenseRepositoryImpl) GetParticipantIDsByItemID(tx *gorm.DB, itemID int64) ([]int64, error) {
	var participantIDs []int64

	err := tx.Model(&entity.ExpenseItemParticipant{}).
		Where("item_id = ?", itemID).
		Pluck("participant_id", &participantIDs).
		Error

	if err != nil {
		return nil, err
	}

	return participantIDs, nil
}

// DeleteItemParticipant implements [ExpenseRepository].
func (e *expenseRepositoryImpl) DeleteItemParticipant(tx *gorm.DB, itemID int64) error {
	return tx.Where("item_id = ?", itemID).Delete(&entity.ExpenseItemParticipant{}).Error
}
