package engine

import (
	"testing"

	"github.com/harkandavez/split-bill-go/internal/helper/money"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

func settlementBalances() []ParticipantBalance {
	return []ParticipantBalance{
		{ParticipantID: 1, Name: "A", NetBalance: money.New(1000, "IDR")},
		{ParticipantID: 2, Name: "B", NetBalance: money.New(-1000, "IDR")},
	}
}

func TestApplySettlementPayments_PartialAndFull(t *testing.T) {
	balances := settlementBalances()
	balances = ApplySettlementPayments(balances, []entity.SettlementPayment{
		{FromParticipantID: 2, ToParticipantID: 1, Amount: 400},
	})

	if balances[0].NetBalance.Amount != 600 || balances[1].NetBalance.Amount != -600 {
		t.Fatalf("partial: got %d/%d, want 600/-600",
			balances[0].NetBalance.Amount, balances[1].NetBalance.Amount)
	}

	balances = ApplySettlementPayments(balances, []entity.SettlementPayment{
		{FromParticipantID: 2, ToParticipantID: 1, Amount: 600},
	})

	if balances[0].NetBalance.Amount != 0 || balances[1].NetBalance.Amount != 0 {
		t.Fatalf("full: got %d/%d, want 0/0",
			balances[0].NetBalance.Amount, balances[1].NetBalance.Amount)
	}
}

func TestApplySettlementPayments_OverpayFlipsToCredit(t *testing.T) {
	balances := ApplySettlementPayments(settlementBalances(), []entity.SettlementPayment{
		{FromParticipantID: 2, ToParticipantID: 1, Amount: 1500},
	})

	if balances[0].NetBalance.Amount != -500 || balances[1].NetBalance.Amount != 500 {
		t.Fatalf("overpay: got %d/%d, want -500/500",
			balances[0].NetBalance.Amount, balances[1].NetBalance.Amount)
	}
}
