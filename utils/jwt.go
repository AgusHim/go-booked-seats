package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWT_SECRET = []byte("secret-key")

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET)
}

// GenerateTicketJWT creates a JWT token for verified ticket holders (war kursi session)
func GenerateTicketJWT(ticketID string, ticketCode string, gender string, category string, ticketName string, name string) (string, error) {
	claims := jwt.MapClaims{
		"ticket_id":   ticketID,
		"ticket_name": ticketName,
		"ticket_code": ticketCode,
		"name":        name,
		"gender":      gender,
		"category":    category,
		"type":        "war_kursi",
		"exp":         time.Now().Add(6 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET)
}
