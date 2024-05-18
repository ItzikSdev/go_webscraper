package main

import (
	"mogodum/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.POST("/scrape_yts", routes.StartScraping)
	router.GET("/", routes.GetScrapeStatus)

	router.Run(":5100")
}
