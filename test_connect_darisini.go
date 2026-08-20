//go:build ignore

// This manual diagnostic is intentionally excluded from normal builds.
// Run it with:
//
//	DARISINI_COOKIE='...' DARISINI_PUBLIC_ID='...' go run test_connect_darisini.go
//
// Never commit a Darisini cookie, scanner password, ticket ID, or response file.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const validateUserTicketQuery = `query useEventScannerValidateUserTicketQuery(
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

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Darisini diagnostic failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cookie := os.Getenv("DARISINI_COOKIE")
	if cookie == "" {
		return errors.New("DARISINI_COOKIE is required")
	}

	publicID := os.Getenv("DARISINI_PUBLIC_ID")
	if len(os.Args) > 1 {
		publicID = os.Args[1]
	}
	if publicID == "" {
		return errors.New("DARISINI_PUBLIC_ID or the first CLI argument is required")
	}

	var eventScannerID *string
	if value := os.Getenv("DARISINI_SCANNER_ID"); value != "" {
		eventScannerID = &value
	}

	var password *string
	if value := os.Getenv("DARISINI_SCANNER_PASSWORD"); value != "" {
		password = &value
	}

	payload, err := json.Marshal(graphQLRequest{
		Query: validateUserTicketQuery,
		Variables: map[string]interface{}{
			"eventScannerId": eventScannerID,
			"password":       password,
			"publicId":       publicID,
		},
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://scanner.darisini.com/api/graphql",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Origin", "https://scanner.darisini.com")
	req.Header.Set("Referer", "https://scanner.darisini.com/v2/presence")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	fmt.Println("STATUS:", resp.Status)

	outputFile := os.Getenv("DARISINI_RESPONSE_FILE")
	if outputFile == "" {
		outputFile = "response.json"
	}
	if err := os.WriteFile(outputFile, body, 0o600); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	fmt.Println("Response written to a local ignored file:", outputFile)
	return nil
}
