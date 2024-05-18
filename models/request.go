package models

type ScrapeRequest struct {
	PAGES int `json:"pages" binding:"required"`
}
