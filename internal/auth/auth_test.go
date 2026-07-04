package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"testing"
	"time"
)

type memStore struct {
	mu   sync.Mutex
	tokens map[string]*RefreshToken
}

func newMemStore() *memStore {
	return &memStore{tokens: make(map[string]*RefreshToken)}
}

func (s *memStore) Create(_ context.Context, userID, tokenHash, familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tokenHash] = &RefreshToken{
		ID:        tokenHash[:8],
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
	}
	return nil
}

func (s *memStore) FindByHash(_ context.Context, hash string) (*RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.tokens[hash]
	return t, nil
}

func (s *memStore) Revoke(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tokens {
		if t.ID == id {
			now := time.Now().UTC()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (s *memStore) RevokeFamily(_ context.Context, familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tokens {
		if t.FamilyID == familyID {
			now := time.Now().UTC()
			t.RevokedAt = &now
		}
	}
	return nil
}

type memDenylist struct {
	mu   sync.Mutex
	set map[string]bool
}

func newMemDenylist() *memDenylist {
	return &memDenylist{set: make(map[string]bool)}
}

func (d *memDenylist) Add(_ context.Context, jti string, _ time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.set[jti] = true
	return nil
}

func (d *memDenylist) IsDenied(_ context.Context, jti string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.set[jti], nil
}

func TestIssueAndVerifyAccessToken(t *testing.T) {
	svc, err := NewService(5*time.Minute, newMemStore(), newMemDenylist())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.IssueAccessToken("user1", []string{"member"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	claims, err := svc.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if claims.UserID != "user1" {
		t.Errorf("expected user_id=user1, got %s", claims.UserID)
	}
}

func TestRejectRevokedToken(t *testing.T) {
	svc, err := NewService(5*time.Minute, newMemStore(), newMemDenylist())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.IssueAccessToken("user1", nil)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	claims, err := svc.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}

	svc.denylist.Add(context.Background(), claims.ID, 5*time.Minute)

	_, err = svc.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("expected error for revoked token, got nil")
	}
}

func TestIssueAndRotateRefreshToken(t *testing.T) {
	svc, err := NewService(5*time.Minute, newMemStore(), newMemDenylist())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	ctx := context.Background()
	token, family, err := svc.IssueRefreshToken(ctx, "user1")
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	rt, err := svc.store.FindByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("find refresh token: %v", err)
	}
	if rt.FamilyID != family {
		t.Errorf("expected family=%s, got %s", family, rt.FamilyID)
	}

	newToken, _, err := svc.RotateRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("rotate refresh token: %v", err)
	}
	if newToken == token {
		t.Error("expected new token different from old")
	}
}

func TestRefreshTokenReuseDetection(t *testing.T) {
	svc, err := NewService(5*time.Minute, newMemStore(), newMemDenylist())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	ctx := context.Background()
	token, _, err := svc.IssueRefreshToken(ctx, "user1")
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	svc.RotateRefreshToken(ctx, token)

	_, err = svc.VerifyRefreshToken(ctx, token)
	if err == nil {
		t.Fatal("expected error for reused token after rotation, got nil")
	}
}

func TestJWKSEndpoint(t *testing.T) {
	svc, err := NewService(5*time.Minute, newMemStore(), newMemDenylist())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	jwks := svc.JWKS()
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}
	if jwks.Keys[0].Alg != "RS256" {
		t.Errorf("expected alg=RS256, got %s", jwks.Keys[0].Alg)
	}
}
