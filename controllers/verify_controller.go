package controllers

import (
	"bytes"
	"fmt"
	"go-ticketing/services"
	"go-ticketing/utils"

	"github.com/gofiber/fiber/v2"
)

type VerifyController struct {
	ticketService services.TicketService
}

func NewVerifyController(ticketService services.TicketService) *VerifyController {
	return &VerifyController{ticketService: ticketService}
}

// VerifyTicket validates a ticket code and returns a JWT token for war kursi access
func (vc *VerifyController) VerifyTicket(c *fiber.Ctx) error {
	type VerifyRequest struct {
		TicketCode string `json:"ticket_code"`
	}

	var body VerifyRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	if body.TicketCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Kode tiket wajib diisi",
		})
	}

	ticket, err := vc.ticketService.VerifyTicketCode(body.TicketCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Kode tiket tidak ditemukan atau tidak valid",
		})
	}

	// Generate JWT token for war kursi session
	token, err := utils.GenerateTicketJWT(ticket.ID, ticket.TicketCode, ticket.Gender, ticket.Category, ticket.TicketName, ticket.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal membuat sesi verifikasi",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Tiket terverifikasi! Anda dapat memilih kursi.",
		"data": fiber.Map{
			"ticket_id":   ticket.ID,
			"ticket_code": ticket.TicketCode,
			"name":        ticket.Name,
			"ticket_name": ticket.TicketName,
			"gender":      ticket.Gender,
			"category":    ticket.Category,
			"token":       token,
			"event_id":    ticket.EventID,
		},
	})
}

// VerifyTicketPDF accepts a PDF e-ticket file, extracts ticket codes via text parsing,
// validates them against the database, and returns verified ticket info with JWT tokens.
func (vc *VerifyController) VerifyTicketPDF(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "File PDF wajib diunggah",
		})
	}

	// Validate file type
	if file.Header.Get("Content-Type") != "application/pdf" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Hanya file PDF yang didukung",
		})
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal membuka file",
		})
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(f); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Gagal membaca file",
		})
	}

	data := buf.Bytes()
	reader := bytes.NewReader(data)

	// Extract ticket codes from PDF
	extracted, err := utils.ExtractTicketsFromPDF(reader, int64(len(data)))
	if err != nil {
		fmt.Println("[PDF OCR] Error extracting:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Gagal membaca PDF: " + err.Error(),
		})
	}

	fmt.Printf("[PDF OCR] Found %d potential tickets in PDF\n", len(extracted))
	for _, ext := range extracted {
		fmt.Printf(" - Extracted Code: '%s'\n", ext.TicketCode)
	}

	if len(extracted) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Tidak ditemukan kode tiket dalam PDF",
		})
	}

	// Validate each extracted ticket against the database
	type VerifiedTicket struct {
		TicketID   string `json:"ticket_id"`
		TicketCode string `json:"ticket_code"`
		Name       string `json:"name"`
		TicketName string `json:"ticket_name"`
		Gender     string `json:"gender"`
		Category   string `json:"category"`
		Token      string `json:"token"`
		OrderID    string `json:"order_id,omitempty"`
		Email      string `json:"email,omitempty"`
		Page       int    `json:"page"`
		EventID    string `json:"event_id"`
	}

	var verifiedTickets []VerifiedTicket

	for _, ext := range extracted {
		ticket, err := vc.ticketService.VerifyTicketCode(ext.TicketCode)
		if err != nil {
			fmt.Printf("[PDF OCR] DB validation failed for '%s': %v\n", ext.TicketCode, err)
			continue // Skip tickets not found in DB
		}

		token, err := utils.GenerateTicketJWT(ticket.ID, ticket.TicketCode, ticket.Gender, ticket.Category, ticket.TicketName, ticket.Name)
		if err != nil {
			continue
		}

		verifiedTickets = append(verifiedTickets, VerifiedTicket{
			TicketID:   ticket.ID,
			TicketCode: ticket.TicketCode,
			Name:       ticket.Name,
			TicketName: ticket.TicketName,
			Gender:     ticket.Gender,
			Category:   ticket.Category,
			Token:      token,
			OrderID:    ext.OrderID,
			Email:      ext.Email,
			Page:       ext.Page,
			EventID:    ticket.EventID,
		})
	}

	if len(verifiedTickets) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success":         false,
			"message":         "Kode tiket dari PDF tidak ditemukan di database",
			"extracted_codes": extractCodes(extracted),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Tiket terverifikasi dari PDF!",
		"data":    verifiedTickets,
	})
}

// extractCodes returns just the ticket codes for debugging
func extractCodes(tickets []utils.ExtractedTicketInfo) []string {
	codes := make([]string, len(tickets))
	for i, t := range tickets {
		codes[i] = t.TicketCode
	}
	return codes
}
