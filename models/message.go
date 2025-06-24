package models

import "encoding/json"

type Message struct {
	Type     string          `json:"type"`
	SenderID string          `json:"sender_id"`
	Payload  json.RawMessage `json:"message"` // bisa di-decode manual berdasarkan `Type`
}
