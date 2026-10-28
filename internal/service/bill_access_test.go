package service

import (
	"testing"
	"time"

	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

func TestEvaluateBillAccess(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	cases := []struct {
		name            string
		bill            entity.Bill
		role            string
		wantPlan        string
		wantPremium     bool
		wantReadOnly    bool
		wantMaxPart     int
		wantMaxExpenses int
	}{
		{
			name:     "group free aktif",
			bill:     entity.Bill{Kind: "group", Status: "ACTIVE", ExpiredAt: &future},
			wantPlan: "free", wantMaxPart: 5, wantMaxExpenses: 15,
		},
		{
			name:     "group free expired",
			bill:     entity.Bill{Kind: "group", Status: "ACTIVE", ExpiredAt: &past},
			wantPlan: "free", wantReadOnly: true, wantMaxPart: 5, wantMaxExpenses: 15,
		},
		{
			name: "group premium expired tetap writable",
			bill: entity.Bill{Kind: "group", Status: "ACTIVE", ExpiredAt: &past},
			role: "premium", wantPlan: "premium", wantPremium: true,
		},
		{
			name:     "goal free aktif",
			bill:     entity.Bill{Kind: "goal", Status: "ACTIVE", ExpiredAt: &future},
			wantPlan: "free", wantMaxPart: 5,
		},
		{
			name:     "split tanpa limit",
			bill:     entity.Bill{Kind: "split", Status: "ACTIVE"},
			wantPlan: "free",
		},
		{
			name:     "status closed read-only",
			bill:     entity.Bill{Kind: "group", Status: "CLOSED"},
			wantPlan: "free", wantReadOnly: true, wantMaxPart: 5, wantMaxExpenses: 15,
		},
	}

	for _, c := range cases {
		got := evaluateBillAccess(&c.bill, c.role)
		if got.Plan != c.wantPlan || got.Premium != c.wantPremium || got.ReadOnly != c.wantReadOnly ||
			got.MaxParticipants != c.wantMaxPart || got.MaxExpenses != c.wantMaxExpenses {
			t.Errorf("%s: evaluateBillAccess() = %+v", c.name, got)
		}
	}
}
