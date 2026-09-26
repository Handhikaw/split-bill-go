package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type ExpenseRepository interface {
	GetCurrencyByExpenseID(ID int64) (string, error)
	FindExpenseByID(tx *gorm.DB, expenseID int64) (*entity.Expense, error)
	UpdateExpense(tx *gorm.DB, expense *entity.Expense) error
	DeleteExpense(tx *gorm.DB, expense *entity.Expense) error
	ReplaceExpenseParticipants(tx *gorm.DB, expenseID int64, participants []entity.ExpenseParticipant) error
	GetExpenseParticipantsByExpenseID(tx *gorm.DB, expenseID int64) ([]entity.ExpenseParticipant, error)
	FindExpenseByBillID(billID int64) ([]entity.Expense, error)
	GetItemsByExpenseID(expenseId int64) ([]entity.ExpenseItem, error)
	GetItemsByExpenseIDs(expenseIds []int64) ([]entity.ExpenseItem, error)
	GetItemParticipantsByItemIDs(itemIds []int64) ([]entity.ExpenseItemParticipant, error)
	GetExpenseParticipantsByExpenseIDs(expenseIds []int64) ([]entity.ExpenseParticipant, error)
	DeleteItemParticipantsByExpenseID(tx *gorm.DB, expenseID int64) error
	DeleteItemsByExpenseID(tx *gorm.DB, expenseID int64) error
	DeleteExpenseParticipantsByExpenseID(tx *gorm.DB, expenseID int64) error
	DeletePaymentsByExpenseID(tx *gorm.DB, expenseID int64) error

	// Expense Item
	CreateItem(tx *gorm.DB, item *entity.ExpenseItem) error
	FindItemByID(tx *gorm.DB, itemID int64) (*entity.ExpenseItem, error)
	UpdateItem(tx *gorm.DB, item *entity.ExpenseItem) error
	DeleteItem(tx *gorm.DB, item *entity.ExpenseItem) error

	// Expense Item Participant
	CreateItemParticipant(tx *gorm.DB, itemParticipant *entity.ExpenseItemParticipant) error
	GetParticipantIDsByItemID(tx *gorm.DB, itemID int64) ([]int64, error)
	DeleteItemParticipant(tx *gorm.DB, itemID int64) error
}
