package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type paymentRepositoryImpl struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepositoryImpl{db: db}
}

// SavePaymentTransaction implements [PaymentRepository].
func (p *paymentRepositoryImpl) SavePaymentTransaction(payment entity.Payment) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		// 1. Upsert Payment (Jika sudah ada untuk expense tersebut, update. Jika belum, create)
		// Kamu bisa sesuaikan logicnya apakah 1 expense bisa multi-payer atau replace
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "expense_id"}, {Name: "paid_by"}},
			DoUpdates: clause.AssignmentColumns([]string{"amount", "paid_at"}),
		}).Create(&payment).Error

		if err != nil {
			return err
		}

		// 2. Update status Bill menjadi 'PAYING'
		// err = tx.Model(&entity.Bill{}).Where("id = ?", p.BillID).
		// 	Update("status", "PAYING").Error

		return err
	})
}

// MovePaymentTransaction memindahkan payer: hapus payment lama lalu
// upsert payment baru dalam satu transaksi.
func (p *paymentRepositoryImpl) MovePaymentTransaction(billID, expenseID, oldPaidBy int64, payment entity.Payment) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("bill_id = ? AND expense_id = ? AND paid_by = ?", billID, expenseID, oldPaidBy).
			Delete(&entity.Payment{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "expense_id"}, {Name: "paid_by"}},
			DoUpdates: clause.AssignmentColumns([]string{"amount", "paid_at"}),
		}).Create(&payment).Error
	})
}

// DeletePayment menghapus satu payment spesifik. Mengembalikan
// gorm.ErrRecordNotFound bila tidak ada yang terhapus.
func (p *paymentRepositoryImpl) DeletePayment(billID, expenseID, paidBy int64) error {
	res := p.db.Where("bill_id = ? AND expense_id = ? AND paid_by = ?", billID, expenseID, paidBy).
		Delete(&entity.Payment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetPaymentsByBillID implements [PaymentRepository].
func (p *paymentRepositoryImpl) GetPaymentsByBillID(billId int64) ([]entity.Payment, error) {
	var payments []entity.Payment
	err := p.db.Where("bill_id = ?", billId).Find(&payments).Error
	return payments, err
}

func (p *paymentRepositoryImpl) ReplaceExpensePayments(
	billID, expenseID int64,
	payments []entity.Payment,
) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bill_id = ? AND expense_id = ?", billID, expenseID).
			Delete(&entity.Payment{}).Error; err != nil {
			return err
		}
		if len(payments) == 0 {
			return nil
		}
		return tx.Create(&payments).Error
	})
}
