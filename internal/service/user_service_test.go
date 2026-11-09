package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"github.com/harkandavez/split-bill-go/internal/service"
	"gorm.io/gorm"
)

type stubUserRepo struct {
	users  map[int64]entity.User
	nextID int64
}

func newStubUserRepo(seed ...entity.User) *stubUserRepo {
	r := &stubUserRepo{users: map[int64]entity.User{}, nextID: 1}
	for _, u := range seed {
		if u.ID >= r.nextID {
			r.nextID = u.ID + 1
		}
		r.users[u.ID] = u
	}
	return r
}

func (r *stubUserRepo) FindByID(id int64) (entity.User, error) {
	u, ok := r.users[id]
	if !ok {
		return entity.User{}, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (r *stubUserRepo) FindByEmail(email string) (entity.User, error) {
	for _, u := range r.users {
		if email != "" && u.Email == email {
			return u, nil
		}
	}
	return entity.User{}, gorm.ErrRecordNotFound
}

func (r *stubUserRepo) FindByAnonID(anonID string) (entity.User, error) {
	for _, u := range r.users {
		if anonID != "" && u.AnonId == anonID {
			return u, nil
		}
	}
	return entity.User{}, gorm.ErrRecordNotFound
}

func (r *stubUserRepo) FindByOAuth(provider, oauthID string) (entity.User, error) {
	for _, u := range r.users {
		if u.OAuthProvider == provider && u.OAuthID == oauthID {
			return u, nil
		}
	}
	return entity.User{}, gorm.ErrRecordNotFound
}

func (r *stubUserRepo) Save(user entity.User) error {
	if user.ID == 0 {
		user.ID = r.nextID
		r.nextID++
	}
	r.users[user.ID] = user
	return nil
}

type stubGoogleVerifier struct {
	identity service.GoogleIdentity
	err      error
}

func (s stubGoogleVerifier) Verify(_ context.Context, _, _ string) (service.GoogleIdentity, error) {
	return s.identity, s.err
}

func newGoogleIdentity() service.GoogleIdentity {
	return service.GoogleIdentity{
		Sub:           "google-sub-1",
		Email:         "budi@example.com",
		Name:          "Budi",
		Picture:       "https://img.example/budi.png",
		EmailVerified: true,
	}
}

func mustAuthGoogle(t *testing.T, repo *stubUserRepo, currentUserID *int64) {
	t.Helper()
	svc := service.NewUserService(repo, "secret", "client-id", stubGoogleVerifier{identity: newGoogleIdentity()})
	res, err := svc.AuthGoogle(context.Background(), "token", currentUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken == "" || res.Provider != "google" || res.Email != "budi@example.com" {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestAuthGoogle_CreateNewUser(t *testing.T) {
	repo := newStubUserRepo()
	mustAuthGoogle(t, repo, nil)

	if len(repo.users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(repo.users))
	}
	for _, u := range repo.users {
		if u.Role != "USER" || u.OAuthID != "google-sub-1" {
			t.Fatalf("unexpected user: %+v", u)
		}
	}
}

func TestAuthGoogle_UpgradeAnonymous(t *testing.T) {
	repo := newStubUserRepo(entity.User{ID: 7, AnonId: "anon-1", Role: "ANONYMOUS"})
	id := int64(7)
	mustAuthGoogle(t, repo, &id)

	u := repo.users[7]
	if len(repo.users) != 1 || u.Role != "USER" || u.OAuthID != "google-sub-1" {
		t.Fatalf("expected anon user upgraded in place, got %+v", repo.users)
	}
}

func TestAuthGoogle_PreservesPremium(t *testing.T) {
	repo := newStubUserRepo(entity.User{ID: 3, Role: "PREMIUM", OAuthProvider: "google", OAuthID: "google-sub-1"})
	svc := service.NewUserService(repo, "secret", "client-id", stubGoogleVerifier{identity: newGoogleIdentity()})
	res, err := svc.AuthGoogle(context.Background(), "token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsPremium || res.Role != "PREMIUM" {
		t.Fatalf("expected premium preserved, got %+v", res)
	}
}

func TestAuthGoogle_InvalidToken(t *testing.T) {
	repo := newStubUserRepo()
	svc := service.NewUserService(repo, "secret", "client-id", stubGoogleVerifier{err: errors.New("bad")})
	if _, err := svc.AuthGoogle(context.Background(), "token", nil); err == nil {
		t.Fatal("expected error for invalid token")
	}
}
