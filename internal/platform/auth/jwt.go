package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var defaultSecret = []byte("bn-merchant-os-dev-secret-key-32bytes-secure!")

// Claims represents the JWT claims payload
type Claims struct {
	UserID     uuid.UUID `json:"sub"`
	Email      string    `json:"email"`
	BusinessID uuid.UUID `json:"biz_id,omitempty"`
	StoreID    uuid.UUID `json:"store_id,omitempty"`
	Role       string    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates an access token for an authenticated user
func GenerateToken(userID uuid.UUID, email string, role string, bizID, storeID uuid.UUID, duration time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		Email:      email,
		BusinessID: bizID,
		StoreID:    storeID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://auth.merchant-os.bookingnetwork.internal",
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{"bn-merchant-os-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(defaultSecret)
}

// ValidateToken parses and validates a JWT string
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return defaultSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
