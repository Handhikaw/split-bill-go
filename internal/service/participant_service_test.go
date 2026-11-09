package service_test

import (
	"testing"

	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type stubActivityService struct {
	logs []entity.ActivityLog
}

func (s *stubActivityService) Log(l entity.ActivityLog) {
	s.logs = append(s.logs, l)
}

func (s *stubActivityService) List(publicID string, limit int) ([]dto.ActivityItem, error) {
	return nil, nil
}

func participantSvc(members map[int64]bool, activity *stubActivityService) service.ParticipantService {
	var activitySvc service.ActivityService
	if activity != nil {
		activitySvc = activity
	}
	return service.NewParticipantService(
		&stubParticipantRepo{members: members},
		&stubBillRepo{bill: testBill()},
		nil,
		activitySvc,
		nil,
		nil,
		nil,
	)
}

// Kasus laporan: header X-Participant-ID ngarang (25 tidak ada di bill)
// wajib 400 sebelum transaksi, bukan 200 + FK error di log.
func TestRemoveParticipant_InvalidActor(t *testing.T) {
	svc := participantSvc(map[int64]bool{26: true}, nil)

	err := svc.RemoveParticipant("bill8", 26, 25)
	if err == nil || err.Error() != "invalid participant" {
		t.Fatalf("expected invalid participant error, got %v", err)
	}
}

func TestUpsertParticipant_InvalidActor(t *testing.T) {
	svc := participantSvc(map[int64]bool{23: true}, nil)

	_, err := svc.UpsertParticipant("bill8", 99, nil, dto.ParticipantRq{Name: "X"})
	if err == nil || err.Error() != "invalid participant" {
		t.Fatalf("expected invalid participant error, got %v", err)
	}
}

// is_owner opsional: tanpa field ini tidak boleh panic.
func TestUpsertParticipant_NilIsOwner(t *testing.T) {
	activity := &stubActivityService{}
	svc := participantSvc(map[int64]bool{23: true}, activity)

	rs, err := svc.UpsertParticipant("bill8", 23, nil, dto.ParticipantRq{Name: "Budi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.Name != "Budi" {
		t.Errorf("unexpected name: %q", rs.Name)
	}
	if len(activity.logs) != 1 || activity.logs[0].ParticipantID != 23 {
		t.Errorf("expected 1 log with actor 23, got %+v", activity.logs)
	}
}

// Mutasi expense dengan actor ngarang juga wajib 400 sebelum transaksi.
func TestDeleteExpense_InvalidActor(t *testing.T) {
	svc := service.NewExpenseService(
		nil,
		&stubBillRepo{bill: testBill()},
		&stubParticipantRepo{members: map[int64]bool{23: true}},
		nil,
		nil,
	)

	err := svc.DeleteExpense("bill8", 17, 99)
	if err == nil || err.Error() != "invalid participant" {
		t.Fatalf("expected invalid participant error, got %v", err)
	}
}
