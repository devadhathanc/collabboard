package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const claimsKey ctxKey = "claims"

func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ah := r.Header.Get("Authorization")
			if ah == "" {
				http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
				return
			}

			tokenStr, ok := strings.CutPrefix(ah, "Bearer ")
			if !ok {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			claims, err := svc.VerifyAccessToken(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(ctx context.Context) *Claims {
	v, _ := ctx.Value(claimsKey).(*Claims)
	return v
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if len(roles) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			hasRole := false
			for _, claimRole := range claims.Roles {
				for _, required := range roles {
					if claimRole == required {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
