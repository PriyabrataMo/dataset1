package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Open the CSV file
	file, err := os.Open("dataMovie.csv") // Replace with your CSV file
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)
	// Read all records from the CSV
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	// Open the output CSV file to write the filtered titles
	outputFile, err := os.Create("filtered_titles.csv")
	if err != nil {
		log.Fatalf("Error creating output file: %v", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// Write header to the output file (optional)
	writer.Write([]string{"title"})

	// Iterate over the records and filter based on the conditions
	for _, record := range records {
		// Skip the header row
		if record[0] == "release_date" {
			continue
		}

		// Parse the release_date to check its validity and year
		releaseDate := record[0]
		if isValidDate(releaseDate) {
			// Check if the year is greater than 1999
			year, err := getYearFromDate(releaseDate)
			if err == nil && year > 2014 && year <= 2024 {
				// Write only the title to the output CSV file
				writer.Write([]string{record[2]})
			}
		}
	}

	fmt.Println("Filtered titles have been saved to 'filtered_titles.csv'")
}

// isValidDate checks if the date is in the format dd/mm/yy
func isValidDate(date string) bool {
	// Try to parse the date in dd/mm/yy format
	_, err := time.Parse("02/01/06", date)
	return err == nil
}

// getYearFromDate extracts the year from a date in dd/mm/yy format
func getYearFromDate(date string) (int, error) {
	parsedDate, err := time.Parse("02/01/06", date)
	if err != nil {
		return 0, err
	}
	return parsedDate.Year(), nil
}
