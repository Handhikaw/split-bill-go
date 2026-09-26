package money

import "testing"

func TestMoney_StringNegative(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{-350000, "IDR", "-350000"},
		{350000, "IDR", "350000"},
		{-1005, "USD", "-10.05"},
		{1005, "USD", "10.05"},
		{-50, "USD", "-0.50"},
		{0, "USD", "0.00"},
	}

	for _, c := range cases {
		if got := New(c.amount, c.currency).String(); got != c.want {
			t.Errorf("New(%d, %s).String() = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}

func TestIsSupportedCurrency(t *testing.T) {
	for _, cur := range []string{"IDR", "JPY", "USD", "EUR", "KWD"} {
		if !IsSupportedCurrency(cur) {
			t.Errorf("expected %s to be supported", cur)
		}
	}
	if IsSupportedCurrency("ABC") {
		t.Errorf("expected ABC to be unsupported")
	}
}
