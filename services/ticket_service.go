// services/ticket_service.go
package services

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TicketService interface {
	Create(ticket *models.Ticket) error
	GetAll(search string, category string, page int, limit int, eventID string) ([]models.Ticket, int64, error)
	GetByID(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
	ImportFromCSV(file multipart.File) error
	VerifyTicketCode(ticketCode string) (*models.Ticket, error)
	ToggleGoodieBag(id string) (*models.Ticket, error)
	MarkGoodieBagsClaimed(ids []string) ([]models.Ticket, error)
	CheckDarisini(id string) (*models.DarisiniTicketCheck, error)
}

type ticketService struct {
	repo        repositories.TicketRepository
	settingRepo repositories.SettingRepository
}

func NewTicketService(repo repositories.TicketRepository, settingRepo ...repositories.SettingRepository) TicketService {
	var settings repositories.SettingRepository
	if len(settingRepo) > 0 {
		settings = settingRepo[0]
	}
	return &ticketService{repo: repo, settingRepo: settings}
}

func (s *ticketService) Create(ticket *models.Ticket) error {
	return s.repo.Create(ticket)
}

func (s *ticketService) GetAll(search string, category string, page int, limit int, eventID string) ([]models.Ticket, int64, error) {
	return s.repo.FindAll(search, category, page, limit, eventID)
}

func (s *ticketService) GetByID(id string) (*models.Ticket, error) {
	return s.repo.FindByID(id)
}

func (s *ticketService) Update(ticket *models.Ticket) error {
	return s.repo.Update(ticket)
}

func (s *ticketService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *ticketService) ImportFromCSV(file multipart.File) error {
	reader := csv.NewReader(file)

	// Skip header
	_, err := reader.Read()
	if err != nil {
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		ticket := models.Ticket{
			ID:          uuid.New().String(), // Optional (bisa juga andalkan BeforeCreate)
			ExtTicketID: record[0],
			TicketCode:  record[0],
			Name:        record[1],
			Email:       record[2],
			Phone:       record[3],
			Gender:      record[4],
			TicketName:  record[5],
			EventID:     record[6],
		}

		if err := s.repo.Create(&ticket); err != nil {
			return err
		}
	}

	return nil
}

func (s *ticketService) VerifyTicketCode(ticketCode string) (*models.Ticket, error) {
	return s.repo.FindByTicketCode(ticketCode)
}

func (s *ticketService) ToggleGoodieBag(id string) (*models.Ticket, error) {
	ticket, err := s.repo.ToggleGoodieBag(id)
	if err != nil {
		return nil, err
	}
	if ticket.GoodieBagClaimed {
		s.scanDarisiniAsync(*ticket)
	}
	return ticket, nil
}

func (s *ticketService) MarkGoodieBagsClaimed(ids []string) ([]models.Ticket, error) {
	tickets, newlyClaimed, err := s.repo.MarkGoodieBagsClaimed(ids)
	if err != nil {
		return nil, err
	}
	for _, ticket := range newlyClaimed {
		s.scanDarisiniAsync(ticket)
	}
	return tickets, nil
}

// validateTicketQuery is the Darisini GraphQL query used to check whether a
// ticket (by publicId) has already been scanned. eventScannerId and password
// are sent as null because authentication is handled via the stored cookie.
const validateTicketQuery = `query useEventScannerValidateUserTicketQuery(
  $eventScannerId: ID
  $password: String
  $publicId: String!
) {
  eventScannerValidateUserTicket(eventScannerId: $eventScannerId, password: $password, publicId: $publicId) {
    success
    error {
      code
      message
      ticketName
      eventTitle
      eventShortUrl
    }
    data {
      publicId
      orderUserEmail
      orderUserFullName
      ownerUserEmail
      ownerUserFullName
      ownerUserGender
      ticket {
        name
        eventTitle
        eventStartDate
      }
      attendance {
        decodedId
        attendedAt
        scannerUserFullName
        notes
        attachmentUrl
      }
      maximumScan
      currentScanCount
    }
  }
}
`

// CheckDarisini looks up a ticket by ID and returns its Darisini validation
// status, including any prior scan (attendance) records.
func (s *ticketService) CheckDarisini(id string) (*models.DarisiniTicketCheck, error) {
	if s.settingRepo == nil {
		return nil, fmt.Errorf("darisini settings not configured")
	}
	ticket, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.validateDarisini(*ticket)
}

// validateDarisini calls the Darisini validate GraphQL query for the given
// ticket using the stored Darisini cookie.
func (s *ticketService) validateDarisini(ticket models.Ticket) (*models.DarisiniTicketCheck, error) {
	cookie, err := s.getDarisiniCookie()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cookie) == "" {
		return nil, fmt.Errorf("darisini cookie is empty")
	}

	publicID := strings.TrimSpace(ticket.TicketCode)
	if publicID == "" {
		publicID = strings.TrimSpace(ticket.ExtTicketID)
	}
	if publicID == "" {
		return nil, fmt.Errorf("ticket code is empty")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": validateTicketQuery,
		"variables": map[string]interface{}{
			"eventScannerId": nil,
			"password":        nil,
			"publicId":        publicID,
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://scanner.darisini.com/api/graphql", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("origin", "https://scanner.darisini.com")
	req.Header.Set("Referer", "https://scanner.darisini.com/v2/presence")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("darisini request failed: HTTP %s", resp.Status)
	}

	var parsed struct {
		Data struct {
			EventScannerValidateUserTicket models.DarisiniTicketCheck `json:"eventScannerValidateUserTicket"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse darisini response: %s", string(body))
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("darisini graphql error: %s", string(body))
	}

	return &parsed.Data.EventScannerValidateUserTicket, nil
}

const createEventAttendanceMutation = `mutation useEventScannerCreateEventAttendanceMutation(
  $input: EventScannerCreateEventAttendanceInput!
) {
  eventScannerCreateEventAttendance(input: $input) {
    id
    decodedId
  }
}
`

func (s *ticketService) scanDarisiniAsync(ticket models.Ticket) {
	if s.settingRepo == nil {
		return
	}

	go func() {
		if err := s.repo.UpdateDarisiniScanLog(ticket.ID, "pending", "Scan Darisini queued", time.Now()); err != nil {
			log.Printf("failed to update darisini pending scan log for ticket %s: %v", ticket.ID, err)
		}
		status, response := s.scanDarisini(ticket)
		if err := s.repo.UpdateDarisiniScanLog(ticket.ID, status, response, time.Now()); err != nil {
			log.Printf("failed to update darisini scan log for ticket %s: %v", ticket.ID, err)
		}
	}()
}

func (s *ticketService) scanDarisini(ticket models.Ticket) (string, string) {
	cookie, err := s.getDarisiniCookie()
	if err != nil {
		return "failed", err.Error()
	}
	if strings.TrimSpace(cookie) == "" {
		return "skipped", "Darisini cookie is empty"
	}

	publicID := strings.TrimSpace(ticket.TicketCode)
	if publicID == "" {
		publicID = strings.TrimSpace(ticket.ExtTicketID)
	}
	if publicID == "" {
		return "skipped", "Ticket code is empty"
	}
	if ticket.Event == nil || strings.TrimSpace(ticket.Event.EventScannerID) == "" {
		return "skipped", "Event Scanner ID is empty"
	}
	if strings.TrimSpace(ticket.Event.EventScannerUserFullName) == "" {
		return "skipped", "Event Scanner user full name is empty"
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": createEventAttendanceMutation,
		"variables": map[string]interface{}{
			"input": map[string]string{
				"publicId":            publicID,
				"eventScannerId":      strings.TrimSpace(ticket.Event.EventScannerID),
				"identityMatch":       "Ticket matches identity",
				"scannerUserFullName": strings.TrimSpace(ticket.Event.EventScannerUserFullName),
				"notes":               "Sudah ambil goodiebag",
			},
		},
	})
	if err != nil {
		return "failed", err.Error()
	}

	req, err := http.NewRequest("POST", "https://scanner.darisini.com/api/graphql", bytes.NewBuffer(payload))
	if err != nil {
		return "failed", err.Error()
	}

	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("origin", "https://scanner.darisini.com")
	req.Header.Set("Referer", "https://scanner.darisini.com/v2/presence")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "failed", err.Error()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "failed", err.Error()
	}

	response := fmt.Sprintf("HTTP %s: %s", resp.Status, string(body))
	if strings.Contains(string(body), "Scan limit reached for this scanner") {
		return "success", response
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "failed", response
	}
	if hasGraphQLErrors(body) {
		return "failed", response
	}
	if strings.Contains(string(body), `"eventScannerCreateEventAttendance"`) {
		return "success", response
	}
	return "failed", response
}

func hasGraphQLErrors(body []byte) bool {
	var response struct {
		Errors []json.RawMessage `json:"errors"`
	}
	return json.Unmarshal(body, &response) == nil && len(response.Errors) > 0
}

func (s *ticketService) getDarisiniCookie() (string, error) {
	setting, err := s.settingRepo.Get(darisiniCookieSettingKey)
	if err != nil || setting == nil {
		return "", err
	}
	return setting.Value, nil
}
