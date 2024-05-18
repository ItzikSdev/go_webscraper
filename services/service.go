package services

import (
	"encoding/json"
	"fmt"
	"log"
	"mogodum/models"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

var (
	taskStatus   string
	statusMutex  sync.Mutex
	movies       []models.Movie
	pagesVisited int
	counter      int
)

func ScrapeYTS(scrapeRequest models.ScrapeRequest) {
	statusMutex.Lock()
	taskStatus = "scraping"
	statusMutex.Unlock()

	baseURL := "https://yts.mx/browse-movies"
	pagesVisited = 0
	counter = 1
	movies = []models.Movie{}
	// movies in yts website = 60440 3022 pages

	for pagesVisited < scrapeRequest.PAGES {
		pageToScrape := fmt.Sprintf("%s?page=%d", baseURL, pagesVisited+1)
		log.Printf("Scraping Page: %d, URL: %s", pagesVisited+1, pageToScrape)
		pageData := PageToScrape(pageToScrape)
		newMovies := ScrapeMovie(pageData, counter, pagesVisited)
		movies = append(movies, newMovies...)
		counter += len(newMovies)
		pagesVisited++
	}

	// Save to JSON file or database as needed
	data, err := json.Marshal(movies)
	if err == nil {
		os.WriteFile("movies.json", data, 0644)
	}

	statusMutex.Lock()
	taskStatus = "completed"
	statusMutex.Unlock()
}

func ScrapeMovie(pageData models.PageData, counter int, pagesVisited int) []models.Movie {
	var movies []models.Movie
	pageData.Doc.Find("ul.tsc_pagination li a[href]")
	pageData.Doc.Find("div.browse-movie-wrap").Each(func(ii int, s *goquery.Selection) {
		movie := models.Movie{
			ID:     uuid.New(),
			Number: counter + pagesVisited,
			URL:    FindHref(s),
			Image:  FindImage(s),
			Name:   FindName(s),
			Year:   FindYear(s),
		}
		inMoviePage := PageToScrape(movie.URL)

		movie.Details = models.Details{
			Genres:     FindGeners(inMoviePage.Doc),
			Likes:      FindLikes(inMoviePage.Doc),
			Rating:     FindRating(inMoviePage.Doc),
			IMDbURL:    FindIMDB(inMoviePage.Doc),
			Summary:    FindSummary(inMoviePage.Doc),
			YoutubeURL: FindYoutubeURL(inMoviePage.Doc),
		}
		movie.Links = FindLinks(inMoviePage.Doc)

		counter++
		movies = append(movies, movie)
		log.Printf("Scraped Movie: %s", movie.Name)
	})
	return movies
}

func PageToScrape(pageToScrape string) models.PageData {
	res, err := http.Get(pageToScrape)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		return models.PageData{Response: res, Doc: nil}
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
		return models.PageData{Response: res, Doc: doc}
	}
	return models.PageData{Response: res, Doc: doc}
}

func FindHref(s *goquery.Selection) string {
	href, exists := s.Find("a.browse-movie-link").Attr("href")
	if exists {
		return href
	}
	return "Error: not find href"
}

func FindImage(s *goquery.Selection) string {
	image, exists := s.Find("img.img-responsive").Attr("src")
	if exists {
		return image
	}
	return "Error: not find image"
}

func FindName(s *goquery.Selection) string {
	title := s.Find("a.browse-movie-title").Text()
	return title
}

func FindYear(s *goquery.Selection) string {
	year := strings.TrimSpace(s.Find("div.browse-movie-year").Text())
	return year
}

func FindGeners(doc *goquery.Document) []string {
	var genres []string

	doc.Find("div.hidden-xs").Each(func(i int, e *goquery.Selection) {
		h2 := e.Find("h2").Text()
		if h2 != "" {
			h2 = strings.TrimSpace(h2)

			// Regex to remove leading year and language tags
			yearPattern := regexp.MustCompile(`^\d+`)
			languageTagPattern := regexp.MustCompile(`\[.*?\]`)

			// Remove the leading year
			h2 = yearPattern.ReplaceAllString(h2, "")
			h2 = strings.TrimSpace(h2)

			// Remove language tags
			h2 = languageTagPattern.ReplaceAllString(h2, "")
			h2 = strings.TrimSpace(h2)

			// Split the remaining string into individual genres
			arrangeGenres := strings.Split(h2, " / ")
			for _, genre := range arrangeGenres {
				// Trim any leading/trailing whitespace from the genre
				genre = strings.TrimSpace(genre)
				genres = append(genres, genre)
			}
		}
	})
	return genres
}

func FindLikes(doc *goquery.Document) float64 {
	var likes float64
	doc.Find("div.bottom-info").Each(func(i int, s *goquery.Selection) {
		movieLikes := s.Find("span#movie-likes").Text()
		if movieLikes == "" {
			log.Printf("Empty likes found")
			return
		}
		float, err := strconv.ParseFloat(strings.ReplaceAll(movieLikes, ",", "."), 64)
		if err != nil {
			log.Printf("Error parsing likes: %v", err)
			return
		}
		likes = float
	})
	return likes
}

func FindRating(doc *goquery.Document) float64 {
	var rating float64
	doc.Find("div.bottom-info").Each(func(i int, s *goquery.Selection) {
		movieRating := s.Find("span[itemprop=ratingValue]").Text()
		float, err := strconv.ParseFloat(strings.ReplaceAll(movieRating, ",", "."), 64)
		if err != nil {
			log.Printf("Error parsing rating: %v", err)
			return
		}
		rating = float
	})
	return rating
}

func FindIMDB(doc *goquery.Document) string {
	var imdbURL string
	imdbLink := doc.Find("a.icon[href*='imdb.com']")
	imdbURL, _ = imdbLink.Attr("href")
	return imdbURL
}

func FindSummary(doc *goquery.Document) string {
	summaryDiv := doc.Find("div#synopsis")
	paragraphs := summaryDiv.Find("p")
	var summary strings.Builder
	paragraphs.Each(func(i int, p *goquery.Selection) {
		// Extract the text content of the <p> element and append it to the summary
		summary.WriteString(p.Text())
		summary.WriteString("\n") // Add a newline between paragraphs
	})
	return summary.String()
}

func FindLinks(doc *goquery.Document) []models.Link {
	var links []models.Link

	doc.Find(".modal-torrent").Each(func(i int, s *goquery.Selection) {
		qualityZise := s.Find("p.quality-size").Text()
		magnetLink := s.Find("a.magnet-download.download-torrent.magnet").AttrOr("href", "")
		magnetTitle := s.Find("a.magnet-download.download-torrent.magnet").AttrOr("title", "")

		link := models.Link{
			QualitySize: qualityZise,
			MagnetLink:  magnetLink,
			MagnetTitle: magnetTitle,
		}
		links = append(links, link)
	})
	return links
}

func FindYoutubeURL(doc *goquery.Document) string {
	var youtubeUrl string
	youtube := doc.Find("a.youtube[href*='youtube.com']")
	youtubeUrl, _ = youtube.Attr("href")
	return youtubeUrl
}

func CreateJsonFile(movies []models.Movie) {
	file, err := os.Create("movies.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Create a JSON encoder and encode the data
	encoder := json.NewEncoder(file)
	err = encoder.Encode(movies)

	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
}
