package engine_test

// import (
// 	"testing"

// 	"github.com/harkandavez/split-bill-go/internal/helper/engine"
// 	"github.com/harkandavez/split-bill-go/internal/model/entity"
// )

// func TestCalculateBalances(t *testing.T) {
// 	// 1. Setup Data sesuai kasusmu
// 	participants := []entity.Participant{
// 		{ID: 1, Name: "A"},
// 		{ID: 2, Name: "B"},
// 	}

// 	expenses := []entity.Expense{
// 		{
// 			ID:       1,
// 			Tax:      3000,
// 			Charge:   5000, // Service
// 			Discount: 10000,
// 		},
// 	}

// 	items := []entity.ExpenseItem{
// 		{ID: 101, ExpenseID: 1, Name: "Ayam A", Price: 30000},
// 		{ID: 102, ExpenseID: 1, Name: "Nasi B", Price: 10000},
// 	}

// 	itemParticipants := []entity.ExpenseItemParticipant{
// 		{ItemID: 101, ParticipantID: 1, ShareAmount: 30000}, // A makan Ayam
// 		{ItemID: 102, ParticipantID: 2, ShareAmount: 10000}, // B makan Nasi
// 	}

// 	// Simulasi Premium: A yang bayar semua (40rb - 10rb disc + 5rb service + 3rb tax = 38.000)
// 	payments := []entity.Payment{
// 		{PaidByParticipantID: 1, Amount: 38000},
// 	}

// 	// 2. Jalankan Engine
// 	balances := engine.CalculateBalances(participants, expenses, items, itemParticipants, payments)

// 	// 3. Verifikasi Hasil
// 	// Porsi A: 30k (makan) - 7.5k (disc) + 3.75k (svc) + 2.25k (tax) = 28.500
// 	// Porsi B: 10k (makan) - 2.5k (disc) + 1.25k (svc) + 0.75k (tax) = 9.500

// 	for _, b := range balances {
// 		if b.ParticipantID == 1 {
// 			// A: Paid 38.000, Owed 28.500 -> Net +9.500 (Nombok)
// 			if b.NetBalance != 9500 {
// 				t.Errorf("A balance salah. Got %d, want 9500", b.NetBalance)
// 			}
// 		}
// 		if b.ParticipantID == 2 {
// 			// B: Paid 0, Owed 9.500 -> Net -9.500 (Utang)
// 			if b.NetBalance != -9500 {
// 				t.Errorf("B balance salah. Got %d, want -9500", b.NetBalance)
// 			}
// 		}
// 	}

// 	// 4. Test Settlement
// 	settlements := engine.GenerateSettlements(balances)
// 	if len(settlements) != 1 {
// 		t.Errorf("Harusnya ada 1 transaksi, dapat %d", len(settlements))
// 	}

// 	s := settlements[0]
// 	if s.From.Name != "B" || s.To.Name != "A" || s.Amount != 9500 {
// 		t.Errorf("Settlement salah: %+v", s)
// 	}
// }

// func TestCalculateBalances_NoPayment(t *testing.T) {
// 	// Setup Data sama seperti sebelumnya
// 	participants := []entity.Participant{
// 		{ID: 1, Name: "A"},
// 		{ID: 2, Name: "B"},
// 	}

// 	expenses := []entity.Expense{
// 		{
// 			ID:       1,
// 			Tax:      3000,
// 			Charge:   5000,
// 			Discount: 10000,
// 		},
// 	}

// 	items := []entity.ExpenseItem{
// 		{ID: 101, ExpenseID: 1, Price: 30000},
// 		{ID: 102, ExpenseID: 1, Price: 10000},
// 	}

// 	itemParticipants := []entity.ExpenseItemParticipant{
// 		{ItemID: 101, ParticipantID: 1, ShareAmount: 30000},
// 		{ItemID: 102, ParticipantID: 2, ShareAmount: 10000},
// 	}

// 	// Payment NIL / Kosong
// 	// var payments []entity.Payment

// 	// 2. Jalankan Engine
// 	balances := engine.CalculateBalances(participants, expenses, items, itemParticipants, nil)

// 	// 3. Verifikasi
// 	for _, b := range balances {
// 		if b.ParticipantID == 1 {
// 			// Owed: 28.500, Paid: 0 -> Net: -28.500
// 			if b.NetBalance != -28500 {
// 				t.Errorf("A balance salah. Got %d, want -28500", b.NetBalance)
// 			}
// 		}
// 		if b.ParticipantID == 2 {
// 			// Owed: 9.500, Paid: 0 -> Net: -9.500
// 			if b.NetBalance != -9500 {
// 				t.Errorf("B balance salah. Got %d, want -9500", b.NetBalance)
// 			}
// 		}
// 	}

// 	// 4. Verifikasi Settlement (Harusnya Kosong)
// 	settlements := engine.GenerateSettlements(balances)
// 	if len(settlements) != 0 {
// 		t.Errorf("Harusnya tidak ada settlement jika tidak ada payer, dapat %d", len(settlements))
// 	}
// }
