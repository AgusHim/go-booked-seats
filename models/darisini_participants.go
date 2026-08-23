package models

import "encoding/json"

// DarisiniParticipantsResponse mirrors the response of the Darisini
// `eventScannerEventUsersConnection` GraphQL query used to pull all
// participants for an event scanner.
type DarisiniParticipantsResponse struct {
	Data struct {
		EventScannerEventUsersConnection DarisiniParticipantsConnection `json:"eventScannerEventUsersConnection"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// DarisiniParticipantsConnection holds the edges (participants) and pagination.
type DarisiniParticipantsConnection struct {
	Edges     []DarisiniParticipantEdge `json:"edges"`
	PageInfo  DarisiniParticipantsPageInfo `json:"pageInfo"`
}

// DarisiniParticipantsPageInfo holds the relay-style pagination cursor.
type DarisiniParticipantsPageInfo struct {
	EndCursor   string `json:"endCursor"`
	HasNextPage bool   `json:"hasNextPage"`
}

// DarisiniParticipantEdge wraps a participant node with its cursor.
type DarisiniParticipantEdge struct {
	Node   DarisiniParticipant `json:"node"`
	Cursor string              `json:"cursor"`
}

// DarisiniParticipant represents a single participant returned by Darisini.
type DarisiniParticipant struct {
	DecodedID    string                         `json:"decodedId"`
	PublicID     string                         `json:"publicId"`
	UserEmail    string                         `json:"userEmail"`
	UserPhone    string                         `json:"userPhone"`
	UserFullName string                         `json:"userFullName"`
	UserGender   string                         `json:"userGender"`
	Ticket       DarisiniParticipantTicket      `json:"ticket"`
	User         DarisiniParticipantUser        `json:"user"`
	Attendance   []DarisiniParticipantAttendance `json:"attendance"`
	ID           string                         `json:"id"`
}

// DarisiniParticipantTicket holds the ticket + event info for a participant.
type DarisiniParticipantTicket struct {
	Name  string `json:"name"`
	Event struct {
		Title string `json:"title"`
		ID    string `json:"id"`
	} `json:"event"`
	ID string `json:"id"`
}

// DarisiniParticipantUser holds the user profile (age, city, province).
type DarisiniParticipantUser struct {
	ID          string `json:"id"`
	UserProfile *struct {
		Age         int    `json:"age"`
		PhoneNumber string `json:"phoneNumber"`
		City        string `json:"city"`
		Province    string `json:"province"`
		ID          string `json:"id"`
	} `json:"userProfile"`
}

// DarisiniParticipantAttendance represents a prior scan record for a participant.
type DarisiniParticipantAttendance struct {
	ScannerUserFullName string   `json:"scannerUserFullName"`
	AttendedAt          string   `json:"attendedAt"`
	Notes               []string `json:"notes"`
	ID                  string   `json:"id"`
}
