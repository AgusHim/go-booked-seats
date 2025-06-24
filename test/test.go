package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash := "$2a$10$2GZISEPQmwNq6Yr/0xA1geGiRzFELpnV9yjOrAhjuaPjboQRcslTO"
	password := "taatbahagia"

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Println("❌ Password tidak cocok:", err)
	} else {
		fmt.Println("✅ Password cocok!")
	}
}
