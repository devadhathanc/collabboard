package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	jwks       JWKS
	accessTTL  time.Duration
	store      RefreshTokenStore
	denylist   Denylist
}

type RefreshTokenStore interface {
	Create(ctx context.Context, userID, tokenHash, familyID string) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeFamily(ctx context.Context, familyID string) error
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	FamilyID  string
	RevokedAt *time.Time
}

type Denylist interface {
	Add(ctx context.Context, jti string, ttl time.Duration) error
	IsDenied(ctx context.Context, jti string) (bool, error)
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewService(accessTTL time.Duration, store RefreshTokenStore, denylist Denylist) (*Service, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	kidBytes := make([]byte, 16)
	rand.Read(kidBytes)
	kid := base64.RawURLEncoding.EncodeToString(kidBytes)

	pub := key.Public().(*rsa.PublicKey)
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString([]byte{0, 0, 0, byte(pub.E)})

	return &Service{
		privateKey: key,
		publicKey:  pub,
		kid:        kid,
		accessTTL:  accessTTL,
		store:      store,
		denylist:   denylist,
		jwks: JWKS{
			Keys: []JWK{{
				Kty: "RSA",
				Kid: kid,
				Use: "sig",
				Alg: "RS256",
				N:   n,
				E:   e,
			}},
		},
	}, nil
}

type Claims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func (s *Service) IssueAccessToken(userID string, roles []string) (string, error) {
	now := time.Now().UTC()
	jti := make([]byte, 16)
	rand.Read(jti)

	claims := Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "collabboard",
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"collabboard-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ID:        base64.RawURLEncoding.EncodeToString(jti),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid
	return token.SignedString(s.privateKey)
}

func (s *Service) VerifyAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	}, jwt.WithLeeway(30*time.Second))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	denied, err := s.denylist.IsDenied(context.Background(), claims.ID)
	if err != nil || denied {
		return nil, fmt.Errorf("token revoked")
	}

	return claims, nil
}

func (s *Service) JWKS() JWKS {
	return s.jwks
}

func (s *Service) IssueRefreshToken(ctx context.Context, userID string) (string, string, error) {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	familyBytes := make([]byte, 16)
	rand.Read(familyBytes)
	familyID := base64.RawURLEncoding.EncodeToString(familyBytes)

	if err := s.store.Create(ctx, userID, tokenHash, familyID); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return token, familyID, nil
}

func (s *Service) VerifyRefreshToken(ctx context.Context, tokenStr string) (*RefreshToken, error) {
	hash := sha256.Sum256([]byte(tokenStr))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	rt, err := s.store.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	if rt.RevokedAt != nil {
		s.store.RevokeFamily(ctx, rt.FamilyID)
		return nil, fmt.Errorf("token revoked")
	}

	return rt, nil
}

func (s *Service) RotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	rt, err := s.VerifyRefreshToken(ctx, oldToken)
	if err != nil {
		return "", "", err
	}

	s.store.Revoke(ctx, rt.ID)

	return s.IssueRefreshToken(ctx, rt.UserID)
}

func (s *Service) RevokeSession(ctx context.Context, tokenID string) error {
	return s.store.Revoke(ctx, tokenID)
}

func (s *Service) RevokeFamily(ctx context.Context, familyID string) error {
	return s.store.RevokeFamily(ctx, familyID)
}

func JSONJWKS(svc *Service) ([]byte, error) {
	return json.Marshal(svc.jwks)
}
