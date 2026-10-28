package service

import (
	"errors"
	"strings"
	"time"

	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

const (
	FreeMaxParticipants = 5
	FreeMaxExpenses     = 15
)

// BillAccess adalah status plan + batas yang berlaku untuk satu bill.
// Sumber premium: role user pembuat bill (bukan role token/login).
type BillAccess struct {
	Plan            string // free | premium
	Premium         bool
	Expired         bool
	ReadOnly        bool
	MaxParticipants int // 0 = unlimited
	MaxExpenses     int // 0 = unlimited
}

func billExpired(bill *entity.Bill) bool {
	if bill.Status != "" && bill.Status != "ACTIVE" {
		return true
	}
	return bill.ExpiredAt != nil && time.Now().After(*bill.ExpiredAt)
}

// evaluateBillAccess murni (tanpa DB) agar mudah dites.
func evaluateBillAccess(bill *entity.Bill, role string) BillAccess {
	kind := bill.Kind
	if kind == "" {
		kind = "group"
	}

	access := BillAccess{Plan: "free"}
	switch kind {
	case "group":
		access.MaxParticipants = FreeMaxParticipants
		access.MaxExpenses = FreeMaxExpenses
	case "goal":
		access.MaxParticipants = FreeMaxParticipants
	}

	if strings.EqualFold(strings.TrimSpace(role), "PREMIUM") {
		access.Premium = true
		access.Plan = "premium"
		access.MaxParticipants = 0
		access.MaxExpenses = 0
	}

	closed := bill.Status != "" && bill.Status != "ACTIVE"
	access.Expired = billExpired(bill)
	// Status CLOSED selalu read-only; expired hanya untuk free.
	access.ReadOnly = closed || (access.Expired && !access.Premium)
	return access
}

// LoadBillAccess mengambil role pembuat bill lalu mengevaluasi batasnya.
// Creator guest (nil) atau user tidak ditemukan dianggap Free.
func LoadBillAccess(db *gorm.DB, bill *entity.Bill) (BillAccess, error) {
	role := ""
	if bill.CreatedBy != nil {
		var user entity.User
		err := db.Select("id, role").Where("id = ?", *bill.CreatedBy).First(&user).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return BillAccess{}, model.NewInternalError("Database error")
		}
		role = user.Role
	}
	return evaluateBillAccess(bill, role), nil
}

// IsPremiumUser mengecek role user (bukan token/login). Nil/guest = false.
func IsPremiumUser(db *gorm.DB, userID *int64) (bool, error) {
	if userID == nil {
		return false, nil
	}
	var user entity.User
	err := db.Select("id, role").Where("id = ?", *userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, model.NewInternalError("Database error")
	}
	return strings.EqualFold(strings.TrimSpace(user.Role), "PREMIUM"), nil
}

func (a BillAccess) ReadOnlyError() error {
	return model.NewBadRequest("masa aktif bill sudah habis; upgrade ke Premium untuk lanjut")
}

func (a BillAccess) ParticipantLimitError() error {
	return model.NewBadRequest("batas peserta Free (5) tercapai; upgrade ke Premium")
}

func (a BillAccess) ExpenseLimitError() error {
	return model.NewBadRequest("batas transaksi Free (15) tercapai; upgrade ke Premium")
}
