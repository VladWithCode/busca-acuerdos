package db

import "time"

type Doc struct {
	ID         string    `json:"id"`
	Case       string    `json:"caseId"`
	Nature     string    `json:"nature"`
	NatureCode string    `json:"natureCode"`
	Accord     string    `json:"accord"`
	AccordDate time.Time `json:"accordDate"`
}
