package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type PaymentRepository interface {
	GetPaymentsByBillID(billId int64) ([]entity.Payment, error)
	SavePaymentTransaction(payment entity.Payment) error
	MovePaymentTransaction(billID, expenseID, oldPaidBy int64, payment entity.Payment) error
	DeletePayment(billID, expenseID, paidBy int64) error
	// ReplaceExpensePayments menghapus semua payer expense lalu insert ulang.
	ReplaceExpensePayments(billID, expenseID int64, payments []entity.Payment) error
}
