package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

type GoogleIdentity struct {
	Sub           string
	Email         string
	Name          string
	Picture       string
	EmailVerified bool
}

type GoogleTokenVerifier interface {
	Verify(ctx context.Context, idToken, audience string) (GoogleIdentity, error)
}

type googleTokenVerifier struct {
	client    *http.Client
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewGoogleTokenVerifier() GoogleTokenVerifier {
	return &googleTokenVerifier{
		client: &http.Client{Timeout: 10 * time.Second},
		keys:   map[string]*rsa.PublicKey{},
	}
}

type googleJWK struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func decodeBase64URL(s string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func (v *googleTokenVerifier) fetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("google certs status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var payload struct {
		Keys []googleJWK `json:"keys"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, k := range payload.Keys {
		if k.Kid == "" || k.N == "" || k.E == "" {
			continue
		}
		nBytes, err := decodeBase64URL(k.N)
		if err != nil {
			continue
		}
		eBytes, err := decodeBase64URL(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		if e == 0 {
			e = 65537
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
	}
	if len(keys) == 0 {
		return errors.New("google certs kosong")
	}
	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

func (v *googleTokenVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	needFetch := len(v.keys) == 0 || time.Since(v.fetchedAt) > time.Hour
	if !needFetch && kid != "" && v.keys[kid] == nil {
		needFetch = true
	}
	if needFetch {
		if err := v.fetchKeys(ctx); err != nil {
			return nil, err
		}
	}

	if kid != "" {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
	}
	if len(v.keys) == 1 {
		for _, key := range v.keys {
			return key, nil
		}
	}
	return nil, errors.New("google signing key tidak ditemukan")
}

func (v *googleTokenVerifier) Verify(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
	if audience == "" {
		return GoogleIdentity{}, errors.New("GOOGLE_CLIENT_ID belum dikonfigurasi")
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		kid, _ := t.Header["kid"].(string)
		return v.key(ctx, kid)
	}, jwt.WithAudience(audience), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return GoogleIdentity{}, fmt.Errorf("google token invalid: %w", err)
	}

	iss, _ := claims["iss"].(string)
	if iss != "https://accounts.google.com" && iss != "accounts.google.com" {
		return GoogleIdentity{}, errors.New("google token issuer tidak valid")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return GoogleIdentity{}, errors.New("google token tanpa sub")
	}
	email, _ := claims["email"].(string)
	verified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return GoogleIdentity{
		Sub:           sub,
		Email:         email,
		Name:          name,
		Picture:       picture,
		EmailVerified: verified,
	}, nil
}
