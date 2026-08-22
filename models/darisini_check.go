package models

// DarisiniTicketCheck mirrors the response of the Darisini
// `eventScannerValidateUserTicket` GraphQL query used to verify whether a
// ticket has already been scanned/validated.
type DarisiniTicketCheck struct {
	Success bool                    `json:"success"`
	Error   *DarisiniCheckError     `json:"error"`
	Data    *DarisiniCheckData      `json:"data"`
}

// DarisiniCheckError holds the optional error returned by Darisini when the
// ticket cannot be validated (e.g. SCAN_LIMIT_REACHED).
type DarisiniCheckError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	TicketName    string `json:"ticketName"`
	EventTitle    string `json:"eventTitle"`
	EventShortURL string `json:"eventShortUrl"`
}

// DarisiniCheckData holds the validated ticket details returned by Darisini.
type DarisiniCheckData struct {
	PublicID          string                `json:"publicId"`
	OrderUserEmail    string                `json:"orderUserEmail"`
	OrderUserFullName string                `json:"orderUserFullName"`
	OwnerUserEmail    string                `json:"ownerUserEmail"`
	OwnerUserFullName string                `json:"ownerUserFullName"`
	OwnerUserGender   string                `json:"ownerUserGender"`
	Ticket            DarisiniCheckTicket   `json:"ticket"`
	Attendance        []DarisiniAttendance  `json:"attendance"`
	MaximumScan       int                   `json:"maximumScan"`
	CurrentScanCount  int                   `json:"currentScanCount"`
}

// DarisiniCheckTicket holds the ticket + event info returned by Darisini.
type DarisiniCheckTicket struct {
	Name           string `json:"name"`
	EventTitle     string `json:"eventTitle"`
	EventStartDate string `json:"eventStartDate"`
}

// DarinisiniAttendance represents a single scan/attendance record for a ticket.
type DarisiniAttendance struct {
	DecodedID           string   `json:"decodedId"`
	AttendedAt          string   `json:"attendedAt"`
	ScannerUserFullName string   `json:"scannerUserFullName"`
	Notes               []string `json:"notes"`
	AttachmentURL       *string  `json:"attachmentUrl"`
}
