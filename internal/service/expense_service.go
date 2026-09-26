package service

import (
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"gorm.io/gorm"
)

type ExpenseService interface {
	CreateExpense(req dto.CreateExpenseRq) (dto.CreateExpenseRs, error)
	UpdateExpense(publicID string, expenseID int64, participantID int64, req dto.UpdateExpenseRq) (dto.UpdateExpenseRs, error)
	DeleteExpense(publicID string, expenseID int64, participantID int64) error
	CreateExpenseItem(publicID string, expenseID int64, participantID int64, req dto.CreateExpenseItemRq) (dto.CreateExpenseItemRs, error)
	UpdateExpenseItem(publicID string, expenseID int64, itemID int64, participantID int64, req dto.UpdateExpenseItemRq) error
	DeleteExpenseItem(publicID string, expenseID int64, itemID int64, participantID int64) error
	GetCategory(userID *int64) ([]dto.CategoryRs, error)
	UpsertCategory(req dto.CategoryRq) error
	SyncExpenseTotal(tx *gorm.DB, expenseID int64) error
}
