package services

import (
	"encoding/json"
	"fmt"
	"log"
	"mogodum/models"
	"os"
)

func CreateJsonFile(jsonNumber int, jsonMovies []models.Movie) {
	filename := fmt.Sprintf("movies/movies%d.json", jsonNumber+1)

	jsonData, err := json.MarshalIndent(jsonMovies, "", "  ")
	if err != nil {
		log.Println("Error marshaling JSON data:", err)
		return
	}

	// If the file already exists, it is truncated. If the file does not exist, it is created with mode 0666
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		log.Println("Error writing JSON data to file:", err)
		return
	}

	log.Printf("Created JSON file %s", filename)
}
