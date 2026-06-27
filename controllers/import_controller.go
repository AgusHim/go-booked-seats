package controllers

import (
	"go-ticketing/models"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportController struct {
	DB *gorm.DB
}

func NewImportController(db *gorm.DB) *ImportController {
	return &ImportController{DB: db}
}

// UploadExcel handles importing participants from .xlsx file
func (ic *ImportController) UploadExcel(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to upload file"})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse excel file"})
	}
	defer f.Close()

	// Assuming data is in the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No sheets found in excel"})
	}
	sheetName := sheets[0]

	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No data found or failed to read rows"})
	}

	headers := rows[0]
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	var newTickets []models.Ticket
	var skippedCount int
	var importedCount int

	eventID := c.FormValue("event_id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "event_id is required"})
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		getCol := func(name string) string {
			idx, ok := headerMap[name]
			if ok && idx < len(row) {
				return row[idx]
			}
			return ""
		}

		ticketCode := getCol("Kode Tiket")
		orderID := getCol("ID Order")

		if ticketCode == "" || orderID == "" {
			continue // Skip invalid rows
		}

		// Check if already exists
		var existingCount int64
		ic.DB.Model(&models.Ticket{}).Where("ticket_code = ? OR order_id = ?", ticketCode, orderID).Count(&existingCount)
		if existingCount > 0 {
			skippedCount++
			continue
		}

		age, _ := strconv.Atoi(getCol("Umur"))

		ticketName := getCol("Tiket")
		category := getCol("Kategori")

		if category == "" {
			lowerTicketName := strings.ToLower(ticketName)
			if strings.Contains(lowerTicketName, "platinum") {
				category = "platinum"
			} else if strings.Contains(lowerTicketName, "gold") {
				category = "gold"
			} else if strings.Contains(lowerTicketName, "silver") {
				category = "silver"
			} else if strings.Contains(lowerTicketName, "vip") {
				category = "vip"
			} else if strings.Contains(lowerTicketName, "reguler") {
				category = "reguler"
			}
		}

		ticket := models.Ticket{
			ID:               uuid.New().String(),
			RegistrationTime: getCol("Waktu Daftar"),
			Name:             getCol("Nama"),
			Email:            getCol("Email"),
			Gender:           getCol("Jenis Kelamin"),
			Phone:            getCol("Telepon"),
			Age:              age,
			City:             getCol("Kota"),
			Province:         getCol("Provinsi"),
			TicketName:       ticketName,
			Category:         category,
			OrderID:          orderID,
			BuyerEmail:       getCol("Email Pemesan"),
			BuyerName:        getCol("Nama pemesan"),
			BuyerPhone:       getCol("Telepon Pemesan"),
			VoucherName:      getCol("Nama voucher"),
			TicketCode:       ticketCode,
			ExtTicketID:      ticketCode,
			EventID:          eventID,
		}
		newTickets = append(newTickets, ticket)
		importedCount++
	}

	if len(newTickets) > 0 {
		// Use CreateInBatches to insert
		err = ic.DB.CreateInBatches(newTickets, 100).Error
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save tickets to database"})
		}
	}

	return c.JSON(fiber.Map{
		"message":        "Import completed",
		"imported_count": importedCount,
		"skipped_count":  skippedCount,
	})
}
