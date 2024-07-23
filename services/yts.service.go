package services

import (
	"fmt"
	"log"
	"mogodum/models"
	"net/http"
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

	for pagesVisited < scrapeRequest.PAGES {
		pageToScrape := fmt.Sprintf("%s?page=%d", baseURL, pagesVisited+1)
		log.Printf("Scraping Page: %d, URL: %s", pagesVisited+1, pageToScrape)
		pageData := PageToScrape(pageToScrape)
		newMovies := ScrapeMovieInYts(pageData, counter, pagesVisited)
		movies = append(movies, newMovies...)
		counter = len(newMovies)
		pagesVisited++
	}

	statusMutex.Lock()
	taskStatus = "completed"
	statusMutex.Unlock()
}

func movieNotFound(movie models.Movie, statusCode int) models.Movie {
	log.Print(movie.Name, statusCode)
	return movie
}

func ScrapeMovieInYts(pageData models.PageData, counter int, pagesVisited int) []models.Movie {
	var movies []models.Movie
	pageData.Doc.Find("ul.tsc_pagination li a[href]")
	pageData.Doc.Find("div.browse-movie-wrap").Each(func(ii int, s *goquery.Selection) {
		movie := models.Movie{
			ID:   uuid.New(),
			URL:  FindHref(s),
			Name: FindName(s),
			Year: FindYear(s),
		}

		moviePage := PageToScrape(movie.URL)

		if moviePage.Response.StatusCode != 200 {
			movie = movieNotFound(movie, moviePage.Response.StatusCode)
		} else {
			movie.Details = models.Details{
				Genres:     FindGeners(moviePage.Doc),
				Likes:      FindLikes(moviePage.Doc),
				Rating:     FindRating(moviePage.Doc),
				IMDbURL:    FindIMDB(moviePage.Doc),
				Summary:    FindSummary(moviePage.Doc),
				YoutubeURL: FindYoutubeURL(moviePage.Doc),
			}

			movieIMDB := PageToScrape(movie.Details.IMDbURL)

			if movieIMDB.Response.StatusCode != 200 {
				movie = movieNotFound(movie, movieIMDB.Response.StatusCode)
			} else {
				movie.DisplayMovies = models.DisplayMovies{
					Name:  movie.Name,
					Image: FindIMDBImage(movieIMDB.Doc),
				}
			}
			movie.Links = FindLinks(moviePage.Doc)
		}
		CreateJsonFile(pagesVisited, movies)
		movies = append(movies, movie)
		log.Printf("Scraped Movie: %s", movie.Name)
		counter++
	})
	return movies
}

func PageToScrape(pageToScrape string) models.PageData {
	req, err := http.NewRequest("GET", pageToScrape, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)

	if res.StatusCode != 200 {
		log.Printf("page: %s, status code error: %d %s", pageToScrape, res.StatusCode, res.Status)
		return models.PageData{Response: res, Doc: doc, URL: pageToScrape}
	}

	if err != nil {
		log.Print(err)
		return models.PageData{Response: res, Doc: doc, URL: pageToScrape, Err: err}
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
	re := regexp.MustCompile(`^\[.*?\]\s*`)
	cleanedTitle := re.ReplaceAllString(title, "")
	return cleanedTitle
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

func FindIMDBImage(doc *goquery.Document) string {
	var imageSrc string
	doc.Find("img.ipc-image").EachWithBreak(func(i int, s *goquery.Selection) bool {
		i++
		if src, exists := s.Attr("src"); exists {
			imageSrc = src
			return false
		}
		return true
	})
	return imageSrc
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
