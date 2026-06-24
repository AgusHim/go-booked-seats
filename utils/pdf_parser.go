package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractedTicketInfo holds parsed ticket data from a PDF page
type ExtractedTicketInfo struct {
	TicketCode string `json:"ticket_code"`
	OrderID    string `json:"order_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`
	Page       int    `json:"page"`
}

// ticketCodeRe matches [ALPHANUMERIC] in square brackets (6-12 chars), allowing newlines
var ticketCodeRe = regexp.MustCompile(`\[\s*([A-Z0-9]{6,12})\s*\]`)

// orderIDRe matches Order #ALPHANUMERIC
var orderIDRe = regexp.MustCompile(`(?i)Order\s*#([A-Z0-9]+)`)

// emailRe matches standard email addresses
var emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// isValidTicketCode checks if the code contains both letters AND digits
// This excludes false positives like ONLINE, OFFLINE, etc.
func isValidTicketCode(code string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range code {
		if c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// ExtractTicketsFromPDF reads a PDF file and extracts e-ticket information from each page.
// Supports multi-page PDFs where each page is a separate e-ticket.
func ExtractTicketsFromPDF(reader io.ReaderAt, size int64) ([]ExtractedTicketInfo, error) {
	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca PDF: %w", err)
	}

	numPages := pdfReader.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF kosong (0 halaman)")
	}

	var tickets []ExtractedTicketInfo
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip pages that can't be read
		}

		ticket := parseTicketText(text, pageNum)
		if ticket != nil && !seen[ticket.TicketCode] {
			seen[ticket.TicketCode] = true
			tickets = append(tickets, *ticket)
		}
	}

	return tickets, nil
}

// ExtractTicketsFromPDFFile is a convenience function that reads from a file path
func ExtractTicketsFromPDFFile(filePath string) ([]ExtractedTicketInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca info file: %w", err)
	}

	return ExtractTicketsFromPDF(f, stat.Size())
}

// parseTicketText extracts ticket info from a single page's text content
func parseTicketText(text string, pageNum int) *ExtractedTicketInfo {
	// Find all bracket matches, pick the first valid ticket code
	matches := ticketCodeRe.FindAllStringSubmatch(text, -1)
	var ticketCode string
	for _, m := range matches {
		if isValidTicketCode(m[1]) {
			ticketCode = m[1]
			break
		}
	}

	if ticketCode == "" {
		return nil
	}

	info := &ExtractedTicketInfo{
		TicketCode: ticketCode,
		Page:       pageNum,
	}

	// Extract Order ID
	if m := orderIDRe.FindStringSubmatch(text); m != nil {
		info.OrderID = m[1]
	}

	// Extract email
	if m := emailRe.FindString(text); m != "" {
		info.Email = m
	}

	// Extract name — line after "Order #..."
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if orderIDRe.MatchString(line) && i+1 < len(lines) {
			candidate := strings.TrimSpace(lines[i+1])
			if candidate != "" && !strings.Contains(candidate, "@") && !strings.Contains(candidate, "http") && len(candidate) > 1 {
				info.Name = candidate
				break
			}
		}
	}

	return info
}
