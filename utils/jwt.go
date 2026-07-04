package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TicketClaims struct {
	TicketID   string
	TicketCode string
	TicketName string
	Name       string
	Gender     string
	Category   string
	EventID    string
}

func JWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Development fallback keeps existing local flows working while production
		// can enforce REQUIRE_JWT_SECRET=true.
		if strings.EqualFold(os.Getenv("REQUIRE_JWT_SECRET"), "true") {
			return nil, errors.New("JWT_SECRET is required")
		}
		secret = "dev-only-change-me-secret"
	}
	if len(secret) < 24 {
		return nil, errors.New("JWT_SECRET must be at least 24 characters")
	}
	return []byte(secret), nil
}

func MustJWTSecret() []byte {
	secret, err := JWTSecret()
	if err != nil {
		panic(err)
	}
	return secret
}

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "admin",
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(MustJWTSecret())
}

// GenerateTicketJWT creates a JWT token for verified ticket holders (war kursi session)
func GenerateTicketJWT(ticketID string, ticketCode string, gender string, category string, ticketName string, name string, eventID string) (string, error) {
	claims := jwt.MapClaims{
		"ticket_id":   ticketID,
		"ticket_name": ticketName,
		"ticket_code": ticketCode,
		"name":        name,
		"gender":      gender,
		"category":    category,
		"event_id":    eventID,
		"type":        "war_kursi",
		"exp":         time.Now().Add(6 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(MustJWTSecret())
}

func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	if tokenStr == "" {
		return nil, errors.New("missing token")
	}

	secret, err := JWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func ClaimString(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok && str != ""
}
