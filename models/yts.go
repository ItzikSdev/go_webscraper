package models

import (
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type Movie struct {
	ID            uuid.UUID     `json:"id"`
	URL           string        `json:"url"`
	Name          string        `json:"name"`
	Year          string        `json:"year"`
	Links         []Link        `json:"links"`
	Details       Details       `json:"details"`
	DisplayMovies DisplayMovies `json:"displayMovies"`
}
type Link struct {
	QualitySize string `json:"quality_size"`
	MagnetLink  string `json:"magnet_link"`
	MagnetTitle string `json:"magnet_title"`
}
type Details struct {
	Genres     []string `json:"genres"`
	Likes      float64  `json:"likes"`
	Rating     float64  `json:"rating"`
	IMDbURL    string   `json:"imdb_url"`
	Summary    string   `json:"summary"`
	YoutubeURL string   `json:"youtube_url"`
}
type PageData struct {
	Response *http.Response
	Doc      *goquery.Document
	URL      string `json:"url"`
	Err      error
}

// create movie
// display all first page
// name, image

// in movie page
// all movie info
