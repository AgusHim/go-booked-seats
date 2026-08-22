package models

type DashboardSummary struct {
	BookedSeats   map[string]map[string]SeatCategorySummary `json:"booked_seats"`
	TicketSummary map[string]map[string]int                 `json:"ticket_summary"`
	GoodieBag     GoodieBagSummary                          `json:"goodie_bag"`
}

type SeatCategorySummary struct {
	TotalSeats  int `json:"total_seats"`
	BookedSeats int `json:"booked_seats"`
	Color       string `json:"color"`
}

// GoodieBagGroupSummary holds the goodie bag claim status breakdown for a
// single grouping (e.g. per ticket category).
type GoodieBagGroupSummary struct {
	Total     int `json:"total"`
	Claimed   int `json:"claimed"`
	Unclaimed int `json:"unclaimed"`
}

// GoodieBagSummary holds the overall goodie bag distribution plus a per-category
// breakdown used to visualize the spread of claimed vs unclaimed goodie bags.
type GoodieBagSummary struct {
	Total      int                              `json:"total"`
	Claimed    int                              `json:"claimed"`
	Unclaimed  int                              `json:"unclaimed"`
	ByCategory map[string]GoodieBagGroupSummary `json:"by_category"`
}
