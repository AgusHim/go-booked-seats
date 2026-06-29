package utils

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

// looseTicketCodeRe matches standalone ticket-code-looking values in QR payloads.
var looseTicketCodeRe = regexp.MustCompile(`\b([A-Z0-9]{6,12})\b`)

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
func ExtractTicketsFromPDF(reader io.ReadSeeker, size int64) ([]ExtractedTicketInfo, error) {
	readerAt, ok := reader.(io.ReaderAt)
	if !ok {
		return nil, fmt.Errorf("reader must implement io.ReaderAt")
	}

	pdfReader, err := pdf.NewReader(readerAt, size)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca PDF: %w", err)
	}

	numPages := pdfReader.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF kosong (0 halaman)")
	}

	var tickets []ExtractedTicketInfo
	seen := make(map[string]bool)

	// Step 1: Text-based Extraction
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

	// Step 2: QR Code Image Extraction using pdfcpu
	if _, err := reader.Seek(0, io.SeekStart); err == nil {
		conf := model.NewDefaultConfiguration()
		if images, err := pdfcpu.ExtractImagesRaw(reader, nil, conf); err == nil {
			qrReader := qrcode.NewQRCodeReader()
			for pageIndex, pageImages := range images {
				pageNum := pageIndex + 1
				for _, img := range pageImages {
					decoded, _, err := image.Decode(img.Reader)
					if err != nil {
						continue
					}
					bmp, err := gozxing.NewBinaryBitmapFromImage(decoded)
					if err != nil {
						continue
					}
					result, err := qrReader.Decode(bmp, nil)
					if err != nil {
						continue
					}
					
					ticketCode := parseTicketCodeFromQRPayload(result.GetText())
					if ticketCode == "" || seen[ticketCode] {
						continue
					}
					seen[ticketCode] = true

					info := ExtractedTicketInfo{TicketCode: ticketCode, Page: pageNum}
					// Attach name/email if parsed from text on the same page
					for _, t := range tickets {
						if t.Page == pageNum {
							info.OrderID = t.OrderID
							info.Name = t.Name
							info.Email = t.Email
							break
						}
					}
					tickets = append(tickets, info)
				}
			}
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

// (Legacy image decoding removed - now using pdfcpu for robust DCTDecode support)

func parseTicketCodeFromQRPayload(payload string) string {
	payload = strings.TrimSpace(strings.ToUpper(payload))
	if payload == "" {
		return ""
	}

	// If the entire payload is exactly a 6-12 character alphanumeric string,
	// it's highly likely to be the ticket code itself.
	if regexp.MustCompile(`^[A-Z0-9]{6,12}$`).MatchString(payload) {
		return payload
	}

	if ticket := parseTicketText(payload, 0); ticket != nil {
		return ticket.TicketCode
	}

	matches := looseTicketCodeRe.FindAllStringSubmatch(payload, -1)
	for _, match := range matches {
		if isValidTicketCode(match[1]) {
			return match[1]
		}
	}

	return ""
}
