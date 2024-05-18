package routes

import (
	"encoding/json"
	"log"
	"mogodum/models"
	"mogodum/services"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	taskStatus   string
	statusMutex  sync.Mutex
	movies       []models.Movie
	pagesVisited int
	counter      int
)

func StartScraping(c *gin.Context) {
	var scrapeRequest models.ScrapeRequest

	if err := c.ShouldBindJSON(&scrapeRequest); err != nil {
		// Handle binding error
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run the scraping task in a new goroutine
	go services.ScrapeYTS(scrapeRequest)

	c.JSON(202, gin.H{"status": "scraping started"})
}

func GetScrapeStatus(c *gin.Context) {
	// Initialize the status
	taskStatus = "idle"
	statusMutex.Lock()
	status := taskStatus
	statusMutex.Unlock()

	// Read the JSON file
	jsonMovies, err := os.ReadFile("movies.json")
	if err != nil {
		log.Println("Error reading movies.json:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	// Unmarshal the JSON data
	var movies []models.Movie
	err = json.Unmarshal(jsonMovies, &movies)
	if err != nil {
		log.Println("Error unmarshaling JSON data:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal JSON data"})
		return
	}
	c.JSON(200, gin.H{"status": status, "movies": movies})
}
