package service

import (
	"testing"
	"time"

	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

func TestBillExpired(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	cases := []struct {
		name string
		bill entity.Bill
		want bool
	}{
		{"active tanpa expiry", entity.Bill{Status: "ACTIVE"}, false},
		{"active belum expired", entity.Bill{Status: "ACTIVE", ExpiredAt: &future}, false},
		{"active sudah expired", entity.Bill{Status: "ACTIVE", ExpiredAt: &past}, true},
		{"status closed", entity.Bill{Status: "CLOSED"}, true},
		{"status kosong tanpa expiry", entity.Bill{}, false},
	}

	for _, c := range cases {
		if got := billExpired(&c.bill); got != c.want {
			t.Errorf("%s: billExpired() = %v, want %v", c.name, got, c.want)
		}
	}
}
