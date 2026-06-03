package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims mirrors the exact structure used in auth-main.go
type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// parseJWT validates the token and returns (orgID, userID, role, error).
// Uses the same jwtKey env var as the auth service.
func parseJWT(tokenStr string) (orgID, userID, role string, err error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtKey), nil
	})
	if err != nil {
		return "", "", "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", "", "", fmt.Errorf("invalid token claims")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return "", "", "", fmt.Errorf("token expired")
	}

	return claims.OrgID, claims.UserID, claims.Role, nil
}

// withClaims stores org/user/role in request context
func withClaims(ctx context.Context, orgID, userID, role string) context.Context {
	ctx = context.WithValue(ctx, ctxOrgID, orgID)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRole, role)
	return ctx
}

// getOrgID pulls org_id from request context (set by authMiddleware)
func getOrgID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxOrgID).(string); ok {
		return v
	}
	return ""
}
