//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	publicID := "MLV8K7QP"
	var eventScannerId *string = nil
	var password *string = nil

	if len(os.Args) > 1 {
		publicID = os.Args[1]
	}
	if len(os.Args) > 2 {
		val := os.Args[2]
		eventScannerId = &val
	}
	if len(os.Args) > 3 {
		val := os.Args[3]
		password = &val
	}

	payload, err := json.Marshal(graphQLRequest{
		Query: validateUserTicketQuery,
		Variables: map[string]interface{}{
			"eventScannerId": eventScannerId,
			"password":       password,
			"publicId":       publicID,
		},
	})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "https://scanner.darisini.com/api/graphql", bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}

	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("origin", "https://scanner.darisini.com")
	req.Header.Set("Referer", "https://scanner.darisini.com/v2/presence")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("Cookie", "__Host-next-auth.csrf-token=c8e11305d489a2f96356c8ba9ad6cc1cb80d51a503497c8b75e4f148307b5a76%7C8c90a3c369ea67984062e5d9967388179d21f0b449a612f3b22f703344fded75; __Secure-next-auth.callback-url=https%3A%2F%2Fscanner.darisini.com%2Fv2%2Flogin%3FscannerId%3Dcmrbwhfeb00x5s601wupxujif; __Secure-next-auth.session-token=eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..tOy6pQ-b8dLRByRe.mrjTq4bsE60ZSCYuMQ8HC3it4GkLYg7cKjxgmBgmSTMFGzs8X5YI-ubsSghieYjiCpSS0l2FqiObJ2pIMFO_uZmW3pZodFGpWGSbunsfflzXjPJfJr-0HOHA9u20pT8q5PoeoKNqyq9vb4rYk5JFbug6WQzrl-y9kZpOE4GJNBrVoRNlowOt0J0cHYiuzik0U7qsg5QYNqqx8mVV1FKnywsA5T-IyG9wEdQ8BfNJgIigyYG008IyZ2LUnb49bULS9TTSCmi6pe87wpyF_uj7NB8g5ZAfBcmNzwZl2N43sdtXWHsCF8VpuJFv3qWYqHkayV9NpYLJNwFZXwYgcJFClU1Gpd0db7RtRaNnyKBOy2nOT08V01Z8MgwP9a9Y6gM.boRW9LFjLwvKkkDx5NsobQ")
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("STATUS:", resp.Status)

	outputFile := "response.json"
	if _, err := os.Stat("go-ticketing"); err == nil {
		outputFile = "go-ticketing/response.json"
	}

	err = os.WriteFile(outputFile, body, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	} else {
		fmt.Println("Response successfully written to", outputFile)
	}
}
