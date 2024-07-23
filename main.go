package main

import (
	"mogodum/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.POST("/scrape_yts", routes.StartScraping) // scraper
	router.GET("/", routes.GetScrapeStatus)          // get json of display movies in start page

	router.Run(":5100")
}
