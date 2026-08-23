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
	ToggleGoodieBag(id string, scannerUserFullName string) (*models.Ticket, error)
	MarkGoodieBagsClaimed(ids []string, scannerUserFullName string) ([]models.Ticket, error)
	CheckDarisini(id string) (*models.DarisiniTicketCheck, error)
	SyncDarisiniParticipants(eventID, eventScannerID string) (int, int, int, error)
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

func (s *ticketService) ToggleGoodieBag(id string, scannerUserFullName string) (*models.Ticket, error) {
	ticket, err := s.repo.ToggleGoodieBag(id)
	if err != nil {
		return nil, err
	}
	if ticket.GoodieBagClaimed {
		s.scanDarisiniAsync(*ticket, scannerUserFullName)
	}
	return ticket, nil
}

func (s *ticketService) MarkGoodieBagsClaimed(ids []string, scannerUserFullName string) ([]models.Ticket, error) {
	tickets, newlyClaimed, err := s.repo.MarkGoodieBagsClaimed(ids)
	if err != nil {
		return nil, err
	}
	for _, ticket := range newlyClaimed {
		s.scanDarisiniAsync(ticket, scannerUserFullName)
	}
	return tickets, nil
}

// validateTicketQuery is the Darisini GraphQL query used to check whether a
// ticket (by publicId) has already been scanned. eventScannerId and password
// are sent as null because authentication is handled via the stored cookie.
// The variable types must be non-null (ID!, String!) to match the positions
// where they are used, even though their values are sent as null.
const validateTicketQuery = `query useEventScannerValidateUserTicketQuery(
  $eventScannerId: ID!
  $password: String!
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
		return nil, fmt.Errorf("darisini request failed: HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
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

func (s *ticketService) scanDarisiniAsync(ticket models.Ticket, scannerUserFullName string) {
	if s.settingRepo == nil {
		return
	}

	go func() {
		if err := s.repo.UpdateDarisiniScanLog(ticket.ID, "pending", "Scan Darisini queued", time.Now()); err != nil {
			log.Printf("failed to update darisini pending scan log for ticket %s: %v", ticket.ID, err)
		}
		status, response := s.scanDarisini(ticket, scannerUserFullName)
		if err := s.repo.UpdateDarisiniScanLog(ticket.ID, status, response, time.Now()); err != nil {
			log.Printf("failed to update darisini scan log for ticket %s: %v", ticket.ID, err)
		}
	}()
}

func (s *ticketService) scanDarisini(ticket models.Ticket, scannerUserFullName string) (string, string) {
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
	scannerName := strings.TrimSpace(scannerUserFullName)
	if scannerName == "" {
		return "skipped", "Scanner user full name is empty"
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": createEventAttendanceMutation,
		"variables": map[string]interface{}{
			"input": map[string]string{
				"publicId":            publicID,
				"eventScannerId":      strings.TrimSpace(ticket.Event.EventScannerID),
				"identityMatch":       "Ticket matches identity",
				"scannerUserFullName": scannerName,
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

// participantsQuery is the Darisini GraphQL query used to pull all participants
// for an event scanner. Variables mirror docs/get-participants-scan.sh
// (eventScannerId & password are non-null per the schema; password is sent as
// null because authentication is handled via the stored cookie). Pagination
// uses the `after` cursor so we can fetch all pages until hasNextPage is false.
const participantsQuery = `query useEventScannerEventUsersConnectionQuery(
  $_eventId: ID
  $keyword: String
  $ticketId: ID
  $orderBy: String
  $scannedByMe: Boolean
  $eventScannerId: ID!
  $password: String!
  $first: Int
  $after: String
) {
  eventScannerEventUsersConnection(_eventId: $_eventId, keyword: $keyword, ticketId: $ticketId, orderBy: $orderBy, scannedByMe: $scannedByMe, eventScannerId: $eventScannerId, password: $password, first: $first, after: $after) {
    edges {
      node {
        decodedId
        publicId
        userEmail
        userPhone
        userFullName
        userGender
        ticket {
          name
          event {
            title
            id
          }
          id
        }
        user {
          userProfile {
            age
            phoneNumber
            city
            province
            id
          }
          id
        }
        attendance {
          scannerUserFullName
          attendedAt
          notes
          id
        }
        id
        __typename
      }
      cursor
    }
    pageInfo {
      endCursor
      hasNextPage
    }
  }
}
`

// SyncDarisiniParticipants fetches all participants for the given event scanner
// from Darisini and upserts them as tickets for the given event. Returns the
// number of imported (new), updated (existing), and skipped rows.
func (s *ticketService) SyncDarisiniParticipants(eventID, eventScannerID string) (int, int, int, error) {
	if s.settingRepo == nil {
		return 0, 0, 0, fmt.Errorf("darisini settings not configured")
	}
	if strings.TrimSpace(eventID) == "" {
		return 0, 0, 0, fmt.Errorf("event id is required")
	}
	if strings.TrimSpace(eventScannerID) == "" {
		return 0, 0, 0, fmt.Errorf("event scanner id is required")
	}

	participants, err := s.fetchDarisiniParticipants(eventScannerID)
	if err != nil {
		return 0, 0, 0, err
	}

	tickets := make([]models.Ticket, 0, len(participants))
	for _, p := range participants {
		code := strings.TrimSpace(p.PublicID)
		if code == "" {
			continue
		}
		age := 0
		city := ""
		province := ""
		if p.User.UserProfile != nil {
			age = p.User.UserProfile.Age
			city = p.User.UserProfile.City
			province = p.User.UserProfile.Province
		}
		tickets = append(tickets, models.Ticket{
			ExtTicketID: code,
			TicketCode:  code,
			OrderID:     strings.TrimSpace(p.DecodedID),
			Name:        p.UserFullName,
			Email:       p.UserEmail,
			Phone:       p.UserPhone,
			Gender:      strings.ToLower(p.UserGender),
			TicketName:  p.Ticket.Name,
			// Category is derived from the ticket name by matching known
			// keyword tiers (platinum, gold, silver).
			Category:  categoryFromTicketName(p.Ticket.Name),
			Age:       age,
			City:      city,
			Province:  province,
		})
	}

	return s.repo.UpsertParticipants(tickets, eventID)
}
// categoryFromTicketName derives the ticket category from the ticket name by
// matching hard-coded keyword tiers. The check is case-insensitive and matches
// on substrings so variations like "Platinum Pass" or "GOLD-EARLY" still work.
func categoryFromTicketName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "platinum"):
		return "Platinum"
	case strings.Contains(lower, "gold"):
		return "Gold"
	case strings.Contains(lower, "silver"):
		return "Silver"
	default:
		return ""
	}
}



// fetchDarisiniParticipants calls the Darisini participants GraphQL query for
// the given event scanner using the stored Darisini cookie. It follows the
// relay `endCursor`/`hasNextPage` pagination until all pages are consumed.
func (s *ticketService) fetchDarisiniParticipants(eventScannerID string) ([]models.DarisiniParticipant, error) {
	cookie, err := s.getDarisiniCookie()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cookie) == "" {
		return nil, fmt.Errorf("darisini cookie is empty")
	}

	const pageSize = 200
	const maxPages = 1000 // hard cap to prevent runaway loops

	var all []models.DarisiniParticipant
	cursor := ""

	for page := 0; page < maxPages; page++ {
		conn, err := s.fetchParticipantsPage(cookie, eventScannerID, cursor, pageSize)
		if err != nil {
			return nil, err
		}
		for _, edge := range conn.Edges {
			all = append(all, edge.Node)
		}
		// Stop when there is no next page or the server returned an empty page.
		if !conn.PageInfo.HasNextPage || len(conn.Edges) == 0 {
			break
		}
		next := strings.TrimSpace(conn.PageInfo.EndCursor)
		if next == "" {
			break
		}
		cursor = next
	}

	return all, nil
}

// fetchParticipantsPage performs a single Darisini participants GraphQL request
// for one page (defined by `first` and `after` cursor) and returns the
// connection (edges + pageInfo) for that page. HTTP errors include the Darisini
// response body so the root cause is visible to the caller.
func (s *ticketService) fetchParticipantsPage(cookie, eventScannerID, after string, first int) (*models.DarisiniParticipantsConnection, error) {
	// Darisini rejects an empty-string cursor ("Invalid cursor: "). For the
	// first page, send `null` instead so the server returns the first page.
	var afterVar interface{}
	if strings.TrimSpace(after) != "" {
		afterVar = after
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": participantsQuery,
		"variables": map[string]interface{}{
			"_eventId":        nil,
			"keyword":         nil,
			"ticketId":        nil,
			"orderBy":         nil,
			"scannedByMe":     nil,
			"eventScannerId":  eventScannerID,
			"password":        nil,
			"first":            first,
			"after":            afterVar,
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
	req.Header.Set("Referer", "https://scanner.darisini.com/v2/participants")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
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
		return nil, fmt.Errorf("darisini request failed: HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed models.DarisiniParticipantsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse darisini response: %s", string(body))
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("darisini graphql error: %s", string(body))
	}

	conn := parsed.Data.EventScannerEventUsersConnection
	return &conn, nil
}
