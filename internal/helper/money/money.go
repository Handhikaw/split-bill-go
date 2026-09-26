package money

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

/*
Money represents an amount in the smallest currency unit.
Examples:
- USD 10.00 -> Amount = 1000
- IDR 25000 -> Amount = 25000
*/
type Money struct {
	Amount   int64  // smallest unit
	Currency string // ISO 4217
}

var currencyScale = map[string]int{
	"IDR": 0,
	"JPY": 0,
	"USD": 2,
	"EUR": 2,
	"KWD": 3,
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// --------------------
// internal helpers
// --------------------

func scale(currency string) int {
	if s, ok := currencyScale[currency]; ok {
		return s
	}
	panic("unsupported currency: " + currency)
}

func IsSupportedCurrency(currency string) bool {
	_, ok := currencyScale[currency]
	return ok
}

func factor(currency string) int64 {
	return int64(math.Pow10(scale(currency)))
}

func percentOf(amount int64, percent float64) int64 {
	// rounding HALF-UP
	return int64(math.Round(float64(amount) * percent / 100))
}

// --------------------
// constructors
// --------------------

func New(amount int64, currency string) Money {
	scale(currency)
	return Money{Amount: amount, Currency: currency}
}

func FromFloat(value float64, currency string) Money {
	f := factor(currency)
	return Money{
		Amount:   int64(math.Round(value * float64(f))),
		Currency: currency,
	}
}

func FromString(value string, currency string) (Money, error) {
	s := scale(currency)
	f := factor(currency)

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return Money{}, fmt.Errorf("invalid money format")
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Money{}, err
	}

	fraction := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > s {
			return Money{}, fmt.Errorf("too many decimal places")
		}

		v, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return Money{}, err
		}

		fraction = v * int64(math.Pow10(s-len(parts[1])))
	}

	return Money{
		Amount:   whole*f + fraction,
		Currency: currency,
	}, nil
}

// --------------------
// basic operations
// --------------------

func (m Money) Add(o Money) Money {
	m.assertSameCurrency(o)
	return Money{Amount: m.Amount + o.Amount, Currency: m.Currency}
}

func (m Money) Sub(o Money) Money {
	m.assertSameCurrency(o)
	return Money{Amount: m.Amount - o.Amount, Currency: m.Currency}
}

func (m Money) Multiply(qty int64) Money {
	return Money{Amount: m.Amount * qty, Currency: m.Currency}
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}

func (m Money) assertSameCurrency(o Money) {
	if m.Currency != o.Currency {
		panic("currency mismatch")
	}
}

// Equal checks if two Money are equal
func (m Money) Equal(o Money) bool {
	m.assertSameCurrency(o)
	return m.Amount == o.Amount
}

// GreaterThan checks if m > o
func (m Money) GreaterThan(o Money) bool {
	m.assertSameCurrency(o)
	return m.Amount > o.Amount
}

// LessThan checks if m < o
func (m Money) LessThan(o Money) bool {
	m.assertSameCurrency(o)
	return m.Amount < o.Amount
}

// GreaterOrEqual checks if m >= o
func (m Money) GreaterOrEqual(o Money) bool {
	m.assertSameCurrency(o)
	return m.Amount >= o.Amount
}

// LessOrEqual checks if m <= o
func (m Money) LessOrEqual(o Money) bool {
	m.assertSameCurrency(o)
	return m.Amount <= o.Amount
}

func (m Money) Negate() Money {
	return Money{
		Amount:   -m.Amount,
		Currency: m.Currency,
	}
}

func (m Money) Min(o Money) Money {
	m.assertSameCurrency(o)

	if m.Amount < o.Amount {
		return m
	}
	return o
}

func (m Money) IsPositive() bool {
	return m.Amount > 0
}

func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// --------------------
// tax / service / discount
// --------------------

// ApplyServiceCharge adds service charge percentage (e.g. 5%)
func (m Money) ApplyServiceCharge(percent float64) Money {
	fee := percentOf(m.Amount, percent)
	return Money{Amount: m.Amount + fee, Currency: m.Currency}
}

// ApplyTax adds tax percentage (e.g. 10%)
func (m Money) ApplyTax(percent float64) Money {
	tax := percentOf(m.Amount, percent)
	return Money{Amount: m.Amount + tax, Currency: m.Currency}
}

// ApplyDiscountFixed subtracts fixed discount
func (m Money) ApplyDiscountFixed(d Money) Money {
	m.assertSameCurrency(d)

	result := max(m.Amount-d.Amount, 0)

	return Money{Amount: result, Currency: m.Currency}
}

// ApplyDiscountPercent subtracts percentage discount
func (m Money) ApplyDiscountPercent(percent float64) Money {
	disc := percentOf(m.Amount, percent)

	result := max(m.Amount-disc, 0)

	return Money{Amount: result, Currency: m.Currency}
}

// --------------------
// split helpers
// --------------------

func (m Money) Split(n int64) []Money {
	if n <= 0 {
		panic("split by zero")
	}

	base := m.Amount / n
	remainder := m.Amount % n

	result := make([]Money, n)
	for i := int64(0); i < n; i++ {
		amt := base
		if i < remainder {
			amt++
		}

		result[i] = Money{Amount: amt, Currency: m.Currency}
	}

	return result
}

func (m Money) SplitRandom(n int64) []Money {
	if n <= 0 {
		panic("split by zero")
	}

	base := m.Amount / n
	remainder := m.Amount % n

	result := make([]Money, n)
	for i := int64(0); i < n; i++ {
		result[i] = Money{
			Amount:   base,
			Currency: m.Currency,
		}
	}

	indexes := make([]int, n)
	for i := 0; i < int(n); i++ {
		indexes[i] = i
	}

	rng.Shuffle(len(indexes), func(i, j int) {
		indexes[i], indexes[j] = indexes[j], indexes[i]
	})

	for i := int64(0); i < remainder; i++ {
		result[indexes[i]].Amount++
	}

	return result
}

// --------------------
// formatting
// --------------------

func (m Money) String() string {
	s := scale(m.Currency)

	if s == 0 {
		return fmt.Sprintf("%d", m.Amount)
	}

	sign := ""
	amt := m.Amount
	if amt < 0 {
		sign = "-"
		amt = -amt
	}

	div := factor(m.Currency)
	whole := amt / div
	fraction := amt % div

	return fmt.Sprintf("%s%d.%0*d", sign, whole, s, fraction)
}
