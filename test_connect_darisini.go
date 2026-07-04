//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	url := "https://www.darisini.com/api/graphql"

	payload := []byte(`{
  "query": "query useEventConnectionQuery(\n  $id: ID\n  $_id: ID\n  $circleId: ID\n  $_circleId: ID\n  $locationType: String\n  $priceType: String\n  $timeType: [String!]\n  $ticketVisibility: String\n  $orderBy: String\n  $keyword: String\n  $isPublished: Boolean\n  $first: Int\n  $category: String\n) {\n  ...useEventConnectionFragmentQuery_Q4ZBX\n}\n\nfragment EventCardItemFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment EventDesktopCardItemFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment EventMobileCardItemFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment NewEventCardFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment NewEventDesktopCardItemFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment NewEventMobileCardItemFragmentQuery on Event {\n  decodedId\n  title\n  startDate\n  endDate\n  isPublished\n  bannerImage {\n    url\n    secureUrl\n    id\n  }\n  locationTypes\n  allowedGender\n  circleProfile {\n    name\n    picture {\n      url\n      secureUrl\n      id\n    }\n    id\n  }\n}\n\nfragment useEventConnectionFragmentQuery_Q4ZBX on Query {\n  eventConnection(locationType: $locationType, priceType: $priceType, timeType: $timeType, ticketVisibility: $ticketVisibility, orderBy: $orderBy, keyword: $keyword, isPublished: $isPublished, circleId: $circleId, _circleId: $_circleId, id: $id, _id: $_id, first: $first, category: $category) {\n    edges {\n      node {\n        id\n        decodedId\n        ...EventMobileCardItemFragmentQuery\n        ...EventDesktopCardItemFragmentQuery\n        ...NewEventMobileCardItemFragmentQuery\n        ...NewEventDesktopCardItemFragmentQuery\n        ...EventCardItemFragmentQuery\n        ...NewEventCardFragmentQuery\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n",
  "variables": {
    "id": null,
    "_id": null,
    "circleId": 
"Q2lyY2xlUHJvZmlsZTpjbDlhcHRiOWEzNzYzNmluZWc1cDR4dXptbg==",
    "_circleId": null,
    "locationType": null,
    "priceType": null,
    "ticketVisibility": null,
    "orderBy": "LATEST",
    "keyword": "",
    "isPublished": false,
    "first": 20,
    "category": null
  }
}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}

	req.Header.Set("accept", "application/graphql-response+json; charset=utf-8, application/json; charset=utf-8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://www.darisini.com")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://www.darisini.com/events")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", `"iOS"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1")

	// Cookie
	req.Header.Set(
		"cookie",
		"ph_phc_rX96fU8eX4FOrxRG71X0wQ0qTeS3C3X1l7EHl7HRAty_posthog=%7B%22%24device_id%22%3A%2201996b81-36c0-7555-8404-fe4bbe0e3c8f%22%2C%22distinct_id%22%3A%2201996b81-36c0-7555-8404-fe4bbe0e3c8f%22%2C%22%24sesid%22%3A%5B1770886050221%2C%22019c5107-32a9-76d7-bcbc-7d20c6786273%22%2C1770885952154%5D%2C%22%24initial_person_info%22%3A%7B%22r%22%3A%22https%3A%2F%2Fwww.google.com%2F%22%2C%22u%22%3A%22https%3A%2F%2Fdarisini.com%2F%22%7D%7D; __Host-next-auth.csrf-token=ae3ef84ab08dc60da57f9233a183568605777878c0069707e1aa9698388f36c6%7C67fef85bc2389285c58a34aae859194346f86d33a401b9e677d0a94d2478e072; __Secure-next-auth.callback-url=https%3A%2F%2Fwww.darisini.com%2F; __Secure-next-auth.session-token=eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..lzmyZVY3Pxr4LTI3.70zESNHf1qk2lK6ZcNXB8mTp7vTFtQ8Y5dKttvloyi1nSFDBvfXkZLR7k797RYBUa8eQ4d6_ANOA-fdM4sDsPDU86mTZAWTQbu1w9wSTXaVDIVlRxBAGmwwN69MdWA6uILa4YFGVBf0T3he6dCyQB-L508jkzX2Rq1SxOyqqdr3CWOBEpZz_hUPf7TxxxDXlZOUM3sEMLhn7CcEJdUeZe6e1I1wpF3mBi2UwxE_gW5wYwQzbEmKDg9qLwyk2PaLgF7InuBVovvxkpZKAS_QH1vi-NQEHE7HhgSPGfDF_4q_LuIlxlumMFB6wem5M3wY9qHqOxowc5QSOvHgmo8hJ-UmjpYAGjYA_ENXtPZhPzFoh8ZqQUUnY5_1oAB3d-jiC5bcFk1xmE-iOyveGZeBko7oYxLo3HVp96Fl-lR9slQz0rHo0qx1oyUyWKS-V6zYudcpiNRlxEk80tDPR4cgLdx4uDEVinJesz_KuGJnAj4B7CoCTh0hL3I2_YrXakFWyvGHS41ecoThHiWxLx550E4YO8jTxvp6kx2RyxezteqCq1JKwnXQ3rbRaB9j-eP3lmFiEazTNV3DZtJdb7XSqgxkNGz3HwmNe0jG0rsc54mMvtRwI2LoHHKT81ICDneoZbDgi9wcwdcVZcEl33L-BxCgviS5w4mGge04TZnREOWsghYOJZZlYAc6eI9tzBv_8rImiZLNgK4Fvl57j_ajCPEffg_OK_XENjCkVrPbwIBxcmtUnAup4E07wOCFr7Fb1SdIOkr5xHhcaVJO9MYIgRVwRJRzFqYPd7zz6LkdJkVpoVwWKIcOPeKNPHTP8jmawbgtNndQMrOPKvVvhXWD0O2dUQCk5orOr2u8ZwqRCZZhU019ZOKG7zIM_0cZAuRBCOSBHEUyBhfoMfggsqCgJSHmYa2I1lIx9wFh9RwmuezldvLd8uLqFcHDhQx98JKTkDgYG3axio5Y_Ngc9MY2_qcXQQG1r1MjeskF6HUUkHBGzTHsX9BgFw_voO-mfAPVDsmvp_OSFcZoF7CmOqddBYLWOIUV-1LznK0UFLjzlcecefp9whYfVKUpedGi1OwOCZvX3LyxJDxZgrgoGvZ1DGfWKMHpPQ7NmD1VdwYxa-SJkoJXj862Bafml3VrH_mmRRHGfWk9tcYMCbkNgnvEdLypJfdrZpr_77mN3K8ct5UEwDrqAuBLxzG8QB5jOypR8vyV6q6JFHheT_9DgAQmv3nqcGlROTxvEAoMCmZhxcSI8UaRdWdhugmDHtzNoZ-9-8CQQ67BFlxj3BxOb4FxDTPIS_i-p_JN4cRqGqH4SEjab9I17bDLuv0czNjB5WK8Nmn6oSAUIN-HAUKlzbuHvKEKG_d7oDKJ_J1DRaHxMvWgbuVtGUjB5.hruXJQ_EcQSDNH_JuuXaZw",
	)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Println("STATUS:", resp.Status)

	err = os.WriteFile("response.json", body, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	} else {
		fmt.Println("Response successfully written to response.json")
	}
}
