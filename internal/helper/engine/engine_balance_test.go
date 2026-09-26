package engine_test

import (
	"testing"

	"github.com/harkandavez/split-bill-go/internal/helper/engine"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

// Kasus nyata: expense SIMPLE "Sewa Mobil" 700rb tanpa item,
// dibagi 23 & 24 @350rb, dibayar 23 sebesar 700rb.
// Harus: owed 350rb masing-masing, settlement 24 -> 23 sebesar 350000.
func TestCalculateBalances_SimpleExpenseWithoutItems(t *testing.T) {
	participants := []entity.Participant{
		{ID: 23, Name: "Handhika"},
		{ID: 24, Name: "Vina"},
	}
	expenses := []entity.Expense{
		{ID: 17, BillID: 8, Title: "Sewa Mobil", Type: "SIMPLE", Amount: 700000},
	}
	expenseParts := []entity.ExpenseParticipant{
		{ID: 1, ExpenseID: 17, ParticipantID: 23, ShareAmount: 350000, ShareType: "EQUAL"},
		{ID: 2, ExpenseID: 17, ParticipantID: 24, ShareAmount: 350000, ShareType: "EQUAL"},
	}
	payments := []entity.Payment{
		{ID: 1, BillID: 8, ExpenseID: 17, PaidBy: 23, Amount: 700000},
	}

	balances, err := engine.CalculateBalances(participants, expenses, nil, nil, expenseParts, payments, "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 2 {
		t.Fatalf("expected 2 balances, got %d", len(balances))
	}

	byID := map[int64]engine.ParticipantBalance{}
	for _, b := range balances {
		byID[b.ParticipantID] = b
	}

	if got := byID[23].TotalOwed.Amount; got != 350000 {
		t.Errorf("participant 23 owed = %d, want 350000", got)
	}
	if got := byID[24].TotalOwed.Amount; got != 350000 {
		t.Errorf("participant 24 owed = %d, want 350000", got)
	}
	if got := byID[23].NetBalance.Amount; got != 350000 {
		t.Errorf("participant 23 net = %d, want 350000", got)
	}
	if got := byID[24].NetBalance.Amount; got != -350000 {
		t.Errorf("participant 24 net = %d, want -350000", got)
	}

	settlements, err := engine.GenerateSettlements(balances)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settlements) != 1 {
		t.Fatalf("expected 1 settlement, got %d", len(settlements))
	}
	s := settlements[0]
	if s.From.ID != 24 || s.To.ID != 23 || s.Amount != "350000" {
		t.Errorf("unexpected settlement: %+v", s)
	}
}

// Expense ber-item harus mengabaikan expense_participant (summary table)
// agar tidak double count.
func TestCalculateBalances_ItemExpenseIgnoresSummaryTable(t *testing.T) {
	participants := []entity.Participant{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
	}
	expenses := []entity.Expense{
		{ID: 1, Title: "Makan", Type: "DETAIL", Tax: 10000},
	}
	items := []entity.ExpenseItem{
		{ID: 11, ExpenseID: 1, Name: "Nasi", Quantity: 2, Price: 25000}, // total 50000
	}
	itemParts := []entity.ExpenseItemParticipant{
		{ID: 1, ItemID: 11, ParticipantID: 1, ShareAmount: 25000},
		{ID: 2, ItemID: 11, ParticipantID: 2, ShareAmount: 25000},
	}
	// Summary yang ditulis SyncExpenseTotal — wajib diabaikan engine.
	expenseParts := []entity.ExpenseParticipant{
		{ID: 1, ExpenseID: 1, ParticipantID: 1, ShareAmount: 25000, ShareType: "EQUAL"},
		{ID: 2, ExpenseID: 1, ParticipantID: 2, ShareAmount: 25000, ShareType: "EQUAL"},
	}

	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, expenseParts, nil, "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// subtotal 50000 + tax 10000 -> 30000 per orang (25000 + 5000).
	for _, b := range balances {
		if got := b.TotalOwed.Amount; got != 30000 {
			t.Errorf("participant %d owed = %d, want 30000", b.ParticipantID, got)
		}
	}
}

// Share milik participant yang sudah tidak ada di bill wajib di-skip.
func TestCalculateBalances_IgnoresGhostParticipantShare(t *testing.T) {
	participants := []entity.Participant{
		{ID: 1, Name: "A"},
	}
	expenses := []entity.Expense{
		{ID: 1, Title: "Makan", Type: "DETAIL"},
	}
	items := []entity.ExpenseItem{
		{ID: 11, ExpenseID: 1, Name: "Nasi", Quantity: 1, Price: 10000},
	}
	itemParts := []entity.ExpenseItemParticipant{
		{ID: 1, ItemID: 11, ParticipantID: 1, ShareAmount: 6000},
		{ID: 2, ItemID: 11, ParticipantID: 999, ShareAmount: 4000}, // ghost
	}

	balances, err := engine.CalculateBalances(participants, expenses, items, itemParts, nil, nil, "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(balances))
	}
	// Hanya 6000 yang dihitung; ghost 4000 tidak ikut subtotal.
	if got := balances[0].TotalOwed.Amount; got != 6000 {
		t.Errorf("owed = %d, want 6000", got)
	}
}

// Tax/diskon negatif (diskon > tax+charge) tetap terdistribusi pas.
func TestCalculateBalances_NegativeExtraDistributedExactly(t *testing.T) {
	participants := []entity.Participant{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}
	expenses := []entity.Expense{
		{ID: 1, Title: "Patungan", Type: "SIMPLE", Amount: 100000, Discount: 1000},
	}
	expenseParts := []entity.ExpenseParticipant{
		{ID: 1, ExpenseID: 1, ParticipantID: 1, ShareAmount: 33334},
		{ID: 2, ExpenseID: 1, ParticipantID: 2, ShareAmount: 33333},
		{ID: 3, ExpenseID: 1, ParticipantID: 3, ShareAmount: 33333},
	}

	balances, err := engine.CalculateBalances(participants, expenses, nil, nil, expenseParts, nil, "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var total int64
	for _, b := range balances {
		total += b.TotalOwed.Amount
	}
	// 100000 - 1000 = 99000 harus pas, tidak ada sen hilang.
	if total != 99000 {
		t.Errorf("total owed = %d, want 99000", total)
	}
}
