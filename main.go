package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
)

// Data represents the structure of the OMDB API response
type Data struct {
	Title       string `json:"Title"`
	Year        string `json:"Year"`
	Rated       string `json:"Rated"`
	ReleaseDate string `json:"Released"`
	Runtime     string `json:"Runtime"`
	Genre       string `json:"Genre"`
	Director    string `json:"Director"`
	Writer      string `json:"Writer"`
	Actors      string `json:"Actors"`
	Language    string `json:"Language"`
	Plot        string `json:"Plot"`
	Response    string `json:"Response"`
	Error       string `json:"Error"`
	IMDbID      string `json:"imdbID"` // Add IMDb ID to fetch the trailer
}

func main() {
	// Start the time tracking
	startTime := time.Now()

	// Load environment variables from .env
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// API Key
	apiKey := os.Getenv("API_KEY")
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY is not set in .env")
	}

	// Read movie names from the input CSV
	file, err := os.Open("movies_2024.csv")
	if err != nil {
		log.Fatalf("Error opening movies_2024.csv: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	movieCount := 0
	const batchSize = 100

	for {
		// Read up to 500 movie titles
		record, err := reader.Read()
		if err == io.EOF || movieCount >= batchSize {
			break
		}
		if err != nil {
			log.Printf("Error reading CSV: %v", err)
			continue
		}

		movieTitle := record[0]
		log.Printf("Querying movie: %s", movieTitle)

		// Query movie details
		data, err := fetchMovieDetails(apiKey, movieTitle)
		if err != nil {
			log.Printf("Error fetching details for %s: %v", movieTitle, err)
			continue
		}

		// Generate YouTube trailer link
		//trailerLink := fmt.Sprintf("https://www.youtube.com/watch?v=%s", data.IMDbID)
		//query := fmt.Sprintf("%s+%s+trailer", data.Title, data.Year)
		trailerLink, err := searchActualYouTubeTrailer(youtubeAPIKey, data.Title, data.Year)
		if err != nil {
			log.Fatal("Error searching YouTube Trailer: %v", err)
		}
		// Save to CSV
		writeToCSV([]string{
			data.Title,
			data.Year,
			data.Rated,
			data.ReleaseDate,
			data.Runtime,
			data.Genre,
			data.Director,
			data.Writer,
			data.Actors,
			data.Language,
			data.Plot,
			trailerLink,
		})

		movieCount++
	}

	// Output the time taken
	duration := time.Since(startTime)
	log.Printf("Processed %d movies in %.2f seconds", movieCount, duration.Seconds())
}

func searchYouTubeFirstTrailer(query string) (string, error) {
	// Construct the search URL
	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	// Create an HTTP request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Add headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.google.com/")

	// Make the HTTP GET request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making GET request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Regex to find video links
	re := regexp.MustCompile(`/watch\?v=[a-zA-Z0-9_-]{11}`)
	match := re.FindString(string(body))

	// Check if a match was found
	if match == "" {
		return "", fmt.Errorf("no video link found")
	}

	// Return the first video link
	return "https://www.youtube.com" + match, nil

}

func searchYouTubeTrailer(title, year string) (string, error) {
	// Construct the YouTube search URL
	searchQuery := fmt.Sprintf("%s+%s+trailer", title, year)
	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(searchQuery)

	// Make the HTTP GET request
	resp, err := http.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch YouTube search results: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Use regex to extract the first video link
	re := regexp.MustCompile(`/watch\?v=[a-zA-Z0-9_-]{11}`)
	match := re.FindString(string(body))

	if match == "" {
		return "", fmt.Errorf("no video link found")
	}

	// Return the full YouTube link
	return "https://www.youtube.com" + match, nil
}

func searchActualYouTubeTrailer(apiKey, title, year string) (string, error) {
	baseURL := "https://www.googleapis.com/youtube/v3/search"
	params := url.Values{}
	params.Add("key", apiKey)
	params.Add("part", "snippet")
	params.Add("q", fmt.Sprintf("%s %s trailer", title, year))
	params.Add("type", "video")
	params.Add("maxResults", "1")

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	response, err := http.Get(requestURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
		} `json:"items"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	if len(result.Items) > 0 {
		return "https://www.youtube.com/watch?v=" + result.Items[0].ID.VideoID, nil
	}

	return "", fmt.Errorf("no trailer found")
}

// fetchMovieDetails queries the OMDB API for a given movie title
func fetchMovieDetails(apiKey, movieTitle string) (Data, error) {
	var data Data

	// Build the API request URL
	baseURL := "https://www.omdbapi.com/"
	params := url.Values{}
	params.Add("apikey", apiKey)
	params.Add("t", movieTitle)
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make the GET request
	response, err := http.Get(requestURL)
	if err != nil {
		return data, fmt.Errorf("error making GET request: %v", err)
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return data, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the JSON response
	err = json.Unmarshal(body, &data)
	if err != nil {
		return data, fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Handle API errors
	if data.Response == "False" {
		return data, fmt.Errorf("OMDB API error: %s", data.Error)
	}

	return data, nil
}

// writeToCSV writes movie details into a CSV file
func writeToCSV(movieDetails []string) {
	// File name
	fileName := "movies.csv"

	// Check if file exists
	fileExists := false
	if _, err := os.Stat(fileName); err == nil {
		fileExists = true
	}

	// Open the file for writing (create if it doesn't exist)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening/creating file: %v", err)
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header if the file is new
	if !fileExists {
		header := []string{"Title", "Year", "Rated", "Release Date", "Runtime", "Genre", "Director", "Writer", "Actors", "Language", "Plot", "Trailer Link"}
		if err := writer.Write(header); err != nil {
			log.Fatalf("Error writing header to CSV: %v", err)
		}
	}

	// Write the movie details
	if err := writer.Write(movieDetails); err != nil {
		log.Fatalf("Error writing movie details to CSV: %v", err)
	}
}
