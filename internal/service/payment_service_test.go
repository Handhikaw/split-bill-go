package service_test

import (
	"errors"
	"testing"

	"github.com/harkandavez/split-bill-go/internal/helper/engine"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/service"
	"gorm.io/gorm"
)

// ---------- stubs ----------

type stubBillRepo struct {
	bill *entity.Bill
}

func (s *stubBillRepo) CreateSplitBill(group *entity.Bill, expense *entity.Expense) error {
	return nil
}
func (s *stubBillRepo) CreateGroupBill(group *entity.Bill) error { return nil }
func (s *stubBillRepo) GetBillByPublicID(publicID string) (*entity.Bill, error) {
	if s.bill == nil || s.bill.PublicID != publicID {
		return nil, gorm.ErrRecordNotFound
	}
	return s.bill, nil
}
func (s *stubBillRepo) CreateParticipant(participant *entity.Participant) error { return nil }

type stubParticipantRepo struct {
	members map[int64]bool // participantID yang ada di bill
}

func (s *stubParticipantRepo) Upsert(participant *entity.Participant) (int64, error) {
	return 0, nil
}
func (s *stubParticipantRepo) FindByID(id int64) (*entity.Participant, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s *stubParticipantRepo) SetUserID(id int64, userID *int64) error { return nil }
func (s *stubParticipantRepo) ExistsByIDAndBillID(participantID int64, billID int64) (bool, error) {
	return s.members[participantID], nil
}
func (s *stubParticipantRepo) Delete(participant *entity.Participant) error { return nil }
func (s *stubParticipantRepo) GetAffectedExpenseIDsByParticipantID(tx *gorm.DB, participantID int64) ([]int64, error) {
	return nil, nil
}
func (s *stubParticipantRepo) DeleteExpenseItemParticipantsByParticipantID(tx *gorm.DB, participantID int64) error {
	return nil
}
func (s *stubParticipantRepo) DeleteExpenseParticipantsByParticipantID(tx *gorm.DB, participantID int64) error {
	return nil
}
func (s *stubParticipantRepo) DeletePaymentsByParticipantID(tx *gorm.DB, participantID int64) error {
	return nil
}
func (s *stubParticipantRepo) UpsertParticipantShare(tx *gorm.DB, ep *entity.ExpenseParticipant) error {
	return nil
}
func (s *stubParticipantRepo) GetParticipants(billId int64) ([]entity.Participant, error) {
	return nil, nil
}

type stubPaymentRepo struct {
	payments []entity.Payment
}

func (s *stubPaymentRepo) GetPaymentsByBillID(billID int64) ([]entity.Payment, error) {
	var out []entity.Payment
	for _, p := range s.payments {
		if p.BillID == billID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (s *stubPaymentRepo) SavePaymentTransaction(payment entity.Payment) error {
	for i, p := range s.payments {
		if p.BillID == payment.BillID && p.ExpenseID == payment.ExpenseID && p.PaidBy == payment.PaidBy {
			s.payments[i] = payment
			return nil
		}
	}
	s.payments = append(s.payments, payment)
	return nil
}
func (s *stubPaymentRepo) MovePaymentTransaction(billID, expenseID, oldPaidBy int64, payment entity.Payment) error {
	found := false
	kept := s.payments[:0]
	for _, p := range s.payments {
		if p.BillID == billID && p.ExpenseID == expenseID && p.PaidBy == oldPaidBy {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return gorm.ErrRecordNotFound
	}
	s.payments = kept
	return s.SavePaymentTransaction(payment)
}
func (s *stubPaymentRepo) DeletePayment(billID, expenseID, paidBy int64) error {
	for i, p := range s.payments {
		if p.BillID == billID && p.ExpenseID == expenseID && p.PaidBy == paidBy {
			s.payments = append(s.payments[:i], s.payments[i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// ---------- helpers ----------

func testBill() *entity.Bill {
	return &entity.Bill{ID: 8, PublicID: "bill8", Name: "Puncak", Currency: "IDR", Status: "ACTIVE"}
}

func paymentSvc(payments []entity.Payment) (service.PaymentService, *stubPaymentRepo) {
	payRepo := &stubPaymentRepo{payments: payments}
	svc := service.NewPaymentService(
		payRepo,
		&stubBillRepo{bill: testBill()},
		&stubParticipantRepo{members: map[int64]bool{23: true, 24: true}},
		nil,
	)
	return svc, payRepo
}

func ptr(i int64) *int64 { return &i }

// ---------- tests ----------

// Kasus user: pindah payer expense 17 dari 23 ke 24 + nominal 750rb.
// Payment lama wajib hilang (bukan nambah baris).
func TestProcessPayment_MovePayer(t *testing.T) {
	svc, payRepo := paymentSvc([]entity.Payment{
		{ID: 1, BillID: 8, ExpenseID: 17, PaidBy: 23, Amount: 700000},
	})

	err := svc.ProcessPayment("bill8", dto.PaymentRq{
		ExpenseID:        17,
		ParticipantID:    24,
		OldParticipantID: ptr(23),
		Amount:           "750000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(payRepo.payments) != 1 {
		t.Fatalf("expected 1 payment row, got %d", len(payRepo.payments))
	}
	got := payRepo.payments[0]
	if got.PaidBy != 24 || got.Amount != 750000 || got.ExpenseID != 17 {
		t.Errorf("unexpected payment row: %+v", got)
	}

	// Settlement mengikuti otomatis: share 375rb/375rb, dibayar 24 penuh.
	participants := []entity.Participant{{ID: 23, Name: "Handhika"}, {ID: 24, Name: "Vina"}}
	expenses := []entity.Expense{{ID: 17, BillID: 8, Title: "Sewa Mobil", Type: "SIMPLE", Amount: 750000}}
	expenseParts := []entity.ExpenseParticipant{
		{ExpenseID: 17, ParticipantID: 23, ShareAmount: 375000},
		{ExpenseID: 17, ParticipantID: 24, ShareAmount: 375000},
	}
	balances, err := engine.CalculateBalances(participants, expenses, nil, nil, expenseParts, payRepo.payments, "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	settlements, err := engine.GenerateSettlements(balances)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settlements) != 1 {
		t.Fatalf("expected 1 settlement, got %d", len(settlements))
	}
	s := settlements[0]
	if s.From.ID != 23 || s.To.ID != 24 || s.Amount != "375000" {
		t.Errorf("unexpected settlement: %+v", s)
	}
}

func TestProcessPayment_MovePayerOldNotFound(t *testing.T) {
	svc, _ := paymentSvc(nil)

	err := svc.ProcessPayment("bill8", dto.PaymentRq{
		ExpenseID:        17,
		ParticipantID:    24,
		OldParticipantID: ptr(23),
		Amount:           "750000",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr interface{ Error() string }
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected ApiError, got %T", err)
	}
	if apiErr.Error() != "old payment not found" {
		t.Errorf("unexpected message: %q", apiErr.Error())
	}
}

// Tanpa old_participant_id = upsert biasa (update nominal payer sama).
func TestProcessPayment_UpsertSamePayer(t *testing.T) {
	svc, payRepo := paymentSvc([]entity.Payment{
		{ID: 1, BillID: 8, ExpenseID: 17, PaidBy: 23, Amount: 700000},
	})

	err := svc.ProcessPayment("bill8", dto.PaymentRq{
		ExpenseID:     17,
		ParticipantID: 23,
		Amount:        "750000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payRepo.payments) != 1 || payRepo.payments[0].Amount != 750000 {
		t.Errorf("expected single updated row, got %+v", payRepo.payments)
	}
}

func TestProcessPayment_MovePayerOldNotParticipant(t *testing.T) {
	svc, payRepo := paymentSvc([]entity.Payment{
		{ID: 1, BillID: 8, ExpenseID: 17, PaidBy: 23, Amount: 700000},
	})

	err := svc.ProcessPayment("bill8", dto.PaymentRq{
		ExpenseID:        17,
		ParticipantID:    24,
		OldParticipantID: ptr(99),
		Amount:           "750000",
	})
	if err == nil || err.Error() != "old_participant_id is not a participant of this bill" {
		t.Fatalf("expected participant validation error, got %v", err)
	}
	if len(payRepo.payments) != 1 {
		t.Errorf("payment rows must be untouched, got %+v", payRepo.payments)
	}
}
