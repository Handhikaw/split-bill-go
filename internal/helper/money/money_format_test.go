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
	for _, cur := range []string{
		"IDR", "JPY", "USD", "EUR", "KWD",
		"SGD", "MYR", "GBP", "THB", "AUD", "CAD", "CNY", "KRW",
	} {
		if !IsSupportedCurrency(cur) {
			t.Errorf("expected %s to be supported", cur)
		}
	}
	if IsSupportedCurrency("ABC") {
		t.Errorf("expected ABC to be unsupported")
	}
}

func TestNewCurrencyScales(t *testing.T) {
	cases := []struct {
		currency string
		amount   int64
		want     string
	}{
		{"SGD", 1050, "10.50"},
		{"MYR", 1050, "10.50"},
		{"GBP", 1005, "10.05"},
		{"THB", 1050, "10.50"},
		{"AUD", 1050, "10.50"},
		{"CAD", 1050, "10.50"},
		{"CNY", 1050, "10.50"},
		{"KRW", 10500, "10500"},
	}

	for _, c := range cases {
		if got := New(c.amount, c.currency).String(); got != c.want {
			t.Errorf("New(%d, %s).String() = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}
