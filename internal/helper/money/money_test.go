package money

import "testing"

func TestNewMoney(t *testing.T) {
	m := New(1000, "USD")

	if m.Amount != 1000 {
		t.Errorf("expected amount 1000, got %d", m.Amount)
	}

	if m.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", m.Currency)
	}
}
func TestNewMoney_UnsupportedCurrency(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unsupported currency")
		}
	}()

	New(1000, "ABC")
}
func TestFromFloat_USD(t *testing.T) {
	m := FromFloat(10.25, "USD")

	if m.Amount != 1025 {
		t.Errorf("expected 1025, got %d", m.Amount)
	}
}
func TestFromFloat_IDR(t *testing.T) {
	m := FromFloat(25000, "IDR")

	if m.Amount != 25000 {
		t.Errorf("expected 25000, got %d", m.Amount)
	}
}
func TestMoneyAdd(t *testing.T) {
	a := New(1000, "USD")
	b := New(500, "USD")

	sum := a.Add(b)

	if sum.Amount != 1500 {
		t.Errorf("expected 1500, got %d", sum.Amount)
	}
}
func TestMoneyAdd_CurrencyMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on currency mismatch")
		}
	}()

	a := New(1000, "USD")
	b := New(1000, "IDR")

	a.Add(b)
}
func TestMoneyMultiply(t *testing.T) {
	m := New(2500, "USD")
	result := m.Multiply(3)

	if result.Amount != 7500 {
		t.Errorf("expected 7500, got %d", result.Amount)
	}
}
func TestMoneySplitEven(t *testing.T) {
	m := New(1000, "USD")
	parts := m.Split(4)

	for _, p := range parts {
		if p.Amount != 250 {
			t.Errorf("expected 250, got %d", p.Amount)
		}
	}
}
func TestMoneySplitWithRemainder(t *testing.T) {
	m := New(1000, "USD")
	parts := m.Split(3)

	expected := []int64{334, 333, 333}

	for i, p := range parts {
		if p.Amount != expected[i] {
			t.Errorf("expected %d, got %d", expected[i], p.Amount)
		}
	}
}
func TestMoneySplitByZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on split by zero")
		}
	}()

	m := New(1000, "USD")
	m.Split(0)
}
func TestMoneyString_USD(t *testing.T) {
	m := New(1667, "USD")

	if m.String() != "16.67" {
		t.Errorf("expected 16.67, got %s", m.String())
	}
}
func TestMoneyString_IDR(t *testing.T) {
	m := New(25000, "IDR")

	if m.String() != "25000" {
		t.Errorf("expected 25000, got %s", m.String())
	}
}
func TestMoneyString_KWD(t *testing.T) {
	m := New(123456, "KWD") // scale 3

	if m.String() != "123.456" {
		t.Errorf("expected 123.456, got %s", m.String())
	}
}
