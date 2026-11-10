package service

import "github.com/harkandavez/split-bill-go/internal/model/dto"

type PaymentService interface {
	ProcessPayment(publicID string, req dto.PaymentRq) error
	ReplaceExpensePayments(publicID string, req dto.ReplacePaymentsRq) error
}
