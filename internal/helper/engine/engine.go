package engine

import (
	"errors"
	"math"
	"sort"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

type ParticipantBalance struct {
	ParticipantID int64
	Name          string
	TotalPaid     money.Money
	TotalOwed     money.Money
	NetBalance    money.Money
}

//
// ===============================
// BALANCE CALCULATION
// ===============================
//

func CalculateBalances(
	participants []entity.Participant,
	expenses []entity.Expense,
	items []entity.ExpenseItem,
	itemParticipants []entity.ExpenseItemParticipant,
	expenseParticipants []entity.ExpenseParticipant,
	payments []entity.Payment,
	currency string,
) ([]ParticipantBalance, error) {

	if currency == "" {
		return nil, errors.New("currency required")
	}

	zero := money.New(0, currency)

	type internalBalance struct {
		paid money.Money
		owed money.Money
	}

	calcMap := make(map[int64]*internalBalance)

	for _, p := range participants {
		calcMap[p.ID] = &internalBalance{
			paid: zero,
			owed: zero,
		}
	}

	// ===============================
	// CREDIT (Payments)
	// ===============================

	for _, pay := range payments {
		if b, ok := calcMap[pay.PaidBy]; ok {
			b.paid = b.paid.Add(money.New(pay.Amount, currency))
		}
	}

	// ===============================
	// DEBIT (Items + proportional charges)
	// Untuk expense yang tidak punya item (SIMPLE), debit diambil dari
	// expense_participant. Expense yang punya item mengabaikan
	// expense_participant karena tabel itu hanya summary dari item shares
	// (di-maintain SyncExpenseTotal) — dibaca dua-duanya = double count.
	// Share milik participant yang sudah tidak ada di bill di-skip agar
	// tidak menggelembungkan subtotal.
	// ===============================

	hasItems := make(map[int64]bool, len(expenses))
	for _, item := range items {
		hasItems[item.ExpenseID] = true
	}

	for _, exp := range expenses {

		var expSubtotal int64
		participantSubtotals := make(map[int64]int64)

		if hasItems[exp.ID] {
			for _, item := range items {
				if item.ExpenseID != exp.ID {
					continue
				}

				for _, ip := range itemParticipants {
					if ip.ItemID != item.ID {
						continue
					}
					if _, ok := calcMap[ip.ParticipantID]; !ok {
						continue
					}
					participantSubtotals[ip.ParticipantID] += ip.ShareAmount
					expSubtotal += ip.ShareAmount
				}
			}
		} else {
			for _, ep := range expenseParticipants {
				if ep.ExpenseID != exp.ID {
					continue
				}
				if _, ok := calcMap[ep.ParticipantID]; !ok {
					continue
				}
				participantSubtotals[ep.ParticipantID] += ep.ShareAmount
				expSubtotal += ep.ShareAmount
			}
		}

		if expSubtotal == 0 {
			continue
		}

		netExtra := exp.Tax + exp.Charge - exp.Discount

		var distributedExtra int64
		pIDs := make([]int64, 0, len(participantSubtotals))
		for id := range participantSubtotals {
			pIDs = append(pIDs, id)
		}

		// stable order biar deterministic
		sort.Slice(pIDs, func(i, j int) bool {
			return pIDs[i] < pIDs[j]
		})

		for i, pID := range pIDs {

			sub := participantSubtotals[pID]
			var shareExtra int64

			if i == len(pIDs)-1 {
				shareExtra = netExtra - distributedExtra
			} else {
				ratio := float64(sub) / float64(expSubtotal)
				shareExtra = int64(math.Round(float64(netExtra) * ratio))
				distributedExtra += shareExtra
			}

			if b, ok := calcMap[pID]; ok {
				b.owed = b.owed.
					Add(money.New(sub, currency)).
					Add(money.New(shareExtra, currency))
			}
		}
	}

	// ===============================
	// BUILD RESULT
	// ===============================

	var results []ParticipantBalance

	for _, p := range participants {
		calc := calcMap[p.ID]

		net := calc.paid.Sub(calc.owed)

		results = append(results, ParticipantBalance{
			ParticipantID: p.ID,
			Name:          p.Name,
			TotalPaid:     calc.paid,
			TotalOwed:     calc.owed,
			NetBalance:    net,
		})
	}

	// ===============================
	// VALIDATION (net must be zero)
	// ===============================

	totalNet := zero
	for _, r := range results {
		totalNet = totalNet.Add(r.NetBalance)
	}

	// if !totalNet.IsZero() {
	// 	return nil, errors.New("balance mismatch: total net is not zero")
	// }

	return results, nil
}

//
// ===============================
// SETTLEMENT GENERATION
// ===============================
//

func GenerateSettlements(
	balances []ParticipantBalance,
) ([]dto.Settlement, error) {

	if len(balances) == 0 {
		return []dto.Settlement{}, nil
	}

	currency := balances[0].NetBalance.Currency
	zero := money.New(0, currency)

	var creditors []ParticipantBalance
	var debtors []ParticipantBalance

	for _, b := range balances {
		if b.NetBalance.GreaterThan(zero) {
			creditors = append(creditors, b)
		} else if b.NetBalance.LessThan(zero) {
			debtors = append(debtors, b)
		}
	}

	// sort biar deterministic
	sort.Slice(creditors, func(i, j int) bool {
		return creditors[i].NetBalance.Amount >
			creditors[j].NetBalance.Amount
	})

	sort.Slice(debtors, func(i, j int) bool {
		return debtors[i].NetBalance.Amount <
			debtors[j].NetBalance.Amount // lebih negatif dulu
	})

	var settlements []dto.Settlement
	i, j := 0, 0

	for i < len(debtors) && j < len(creditors) {

		debt := debtors[i].NetBalance.Negate()
		credit := creditors[j].NetBalance

		amount := debt.Min(credit)

		if amount.IsPositive() {
			settlements = append(settlements, dto.Settlement{
				From: dto.PersonInfo{
					ID:   debtors[i].ParticipantID,
					Name: debtors[i].Name,
				},
				To: dto.PersonInfo{
					ID:   creditors[j].ParticipantID,
					Name: creditors[j].Name,
				},
				Amount: amount.String(),
			})
		}

		debtors[i].NetBalance =
			debtors[i].NetBalance.Add(amount)

		creditors[j].NetBalance =
			creditors[j].NetBalance.Sub(amount)

		if debtors[i].NetBalance.Equal(zero) {
			i++
		}
		if creditors[j].NetBalance.Equal(zero) {
			j++
		}
	}

	return settlements, nil
}

//
// ===============================
// DTO MAPPING
// ===============================
//

func ToParticipantBalanceDTO(p ParticipantBalance) dto.ParticipantBalance {
	return dto.ParticipantBalance{
		ParticipantID: p.ParticipantID,
		Name:          p.Name,
		TotalPaid:     p.TotalPaid.String(),
		TotalOwed:     p.TotalOwed.String(),
		NetBalance:    p.NetBalance.String(),
	}
}

//
// ===============================
// APPLY SETTLEMENT PAYMENTS
// ===============================
//

// ApplySettlementPayments mengurangi utang yang sudah dibayar.
// from membayar to: net from naik, net to turun. Overpay boleh;
// hasilnya bisa berbalik menjadi kredit (engine tetap konsisten).
func ApplySettlementPayments(
	balances []ParticipantBalance,
	payments []entity.SettlementPayment,
) []ParticipantBalance {
	if len(balances) == 0 || len(payments) == 0 {
		return balances
	}

	currency := balances[0].NetBalance.Currency
	idx := make(map[int64]int, len(balances))
	for i, b := range balances {
		idx[b.ParticipantID] = i
	}

	for _, p := range payments {
		if p.Amount <= 0 {
			continue
		}
		amount := money.New(p.Amount, currency)
		if i, ok := idx[p.FromParticipantID]; ok {
			balances[i].NetBalance = balances[i].NetBalance.Add(amount)
		}
		if i, ok := idx[p.ToParticipantID]; ok {
			balances[i].NetBalance = balances[i].NetBalance.Sub(amount)
		}
	}

	return balances
}
