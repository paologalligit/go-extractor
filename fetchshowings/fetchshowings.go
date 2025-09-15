package fetchshowings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/utils"
)

type FetchShowingsOptions struct {
	MaxGoroutines  int
	RequestDelay   int
	ShowingUrl     string
	OutputFileName string
	CookiesManager *header.CookiesManager
}

// RunFetchShowings fetches showings and writes them to a file
func RunFetchShowings(options *FetchShowingsOptions) error {
	if err := fetchCinemas(options.CookiesManager); err != nil {
		return fmt.Errorf("failed to fetch cinemas: %w", err)
	}
	fmt.Println("🏠 Cinemas fetched")
	if err := fetchFilms(options.CookiesManager); err != nil {
		return fmt.Errorf("failed to fetch films: %w", err)
	}
	fmt.Println("🎬 Films fetched")

	// Read cinema and film data
	cinemaIds, regionData, err := getCinemaIds()
	if err != nil {
		return fmt.Errorf("failed to get cinema ids: %w", err)
	}
	filmIds, err := getFilmIds()
	if err != nil {
		return fmt.Errorf("failed to get film ids: %w", err)
	}

	// Create work items (combinations of cinema and film IDs)
	var workItems []entities.WorkItem
	for _, cinemaId := range cinemaIds {
		for _, filmId := range filmIds {
			workItems = append(workItems, entities.WorkItem{CinemaId: cinemaId, FilmId: filmId})
		}
	}

	totalRequests := len(workItems)
	fmt.Printf("Total requests to make: %d\n", totalRequests)

	// Create channels for work distribution and result collection
	jobs := make(chan entities.WorkItem, totalRequests)
	results := make(chan entities.ShowingResult, totalRequests)

	// Counter for completed requests
	var completed int64 = 0

	// Start progress reporting in a separate goroutine
	stopProgress := make(chan struct{})
	go reportProgress(&completed, int64(totalRequests), stopProgress)

	// Start worker goroutines
	var wg sync.WaitGroup

	// Limit to maxGoroutines or total work items, whichever is smaller
	workerCount := min(options.MaxGoroutines, totalRequests)
	fmt.Printf("👷 Starting %d workers\n", workerCount)

	for range workerCount {
		wg.Add(1)
		go worker(jobs, results, &wg, regionData, options.RequestDelay, &completed, options)
	}

	// Add jobs to channel
	for _, item := range workItems {
		jobs <- item
	}
	close(jobs)

	// Wait for all goroutines to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
		close(stopProgress) // Stop the progress reporting
	}()

	// Collect results
	var finalResults []entities.ShowingResult
	for result := range results {
		finalResults = append(finalResults, result)
	}

	// Write results to file
	if err := utils.WriteResultsToFile(finalResults, options.OutputFileName); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	fmt.Println("\n🏁 Done! Results written to", options.OutputFileName, ".json")
	return nil
}

func fetchCinemas(cookiesManager *header.CookiesManager) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", constant.CINEMAS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for cinemas: %w", err)
	}

	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return fmt.Errorf("failed to get headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch cinemas: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for cinemas: %w", err)
	}

	var cinemas entities.CinemasFile
	if err := json.Unmarshal(body, &cinemas); err != nil {
		return fmt.Errorf("failed to parse cinemas: %w", err)
	}

	os.WriteFile("cinemas.json", body, 0644)
	return nil
}

func fetchFilms(cookiesManager *header.CookiesManager) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", constant.FILMS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for films: %w", err)
	}

	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return fmt.Errorf("failed to get headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch films: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for films: %w", err)
	}

	var films entities.FilmsFile
	if err := json.Unmarshal(body, &films); err != nil {
		return fmt.Errorf("failed to parse films: %w", err)
	}

	os.WriteFile("films.json", body, 0644)
	return nil
}

func getCinemaIds() ([]string, []entities.Region, error) {
	file, err := os.Open("cinemas.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cinemas.json: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read cinemas.json: %w", err)
	}

	var cinemas entities.CinemasFile
	if err = json.Unmarshal(data, &cinemas); err != nil {
		return nil, nil, fmt.Errorf("failed to parse cinemas.json: %w", err)
	}

	var cinemaIds []string
	for _, reg := range cinemas.Result {
		for _, cinema := range reg.Cinemas {
			cinemaIds = append(cinemaIds, cinema.CinemaId)
		}
	}

	return cinemaIds, cinemas.Result, nil
}

func getFilmIds() ([]string, error) {
	file, err := os.Open("films.json")
	if err != nil {
		return nil, fmt.Errorf("failed to open films.json: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read films.json: %w", err)
	}

	var films entities.FilmsFile
	if err := json.Unmarshal(data, &films); err != nil {
		return nil, fmt.Errorf("failed to parse films.json: %w", err)
	}

	var filmIds []string
	for _, film := range films.Result {
		filmIds = append(filmIds, film.FilmId)
	}

	return filmIds, nil
}

func reportProgress(completed *int64, total int64, stop chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			current := atomic.LoadInt64(completed)
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rProgress: %d/%d (%.2f%%) completed", current, total, percent)
		case <-stop:
			// Final progress update
			current := atomic.LoadInt64(completed)
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rProgress: %d/%d (%.2f%%) completed", current, total, percent)
			return
		}
	}
}

func worker(jobs <-chan entities.WorkItem, results chan<- entities.ShowingResult, wg *sync.WaitGroup, regionData []entities.Region, requestDelay int, completed *int64, options *FetchShowingsOptions) {
	defer wg.Done()

	for job := range jobs {
		result, err := fetchShowing(job.CinemaId, job.FilmId, regionData, options.CookiesManager, options)
		if err != nil {
			fmt.Printf("\nError fetching showing for cinema %s, film %s: %v\n", job.CinemaId, job.FilmId, err)
			continue
		}
		if result.FilmId == "" {
			continue
		}

		booking, err := fetchAllSeats(job.CinemaId, &result, options.CookiesManager)
		if err != nil {
			fmt.Printf("\nError fetching booking for cinema %s, film %s: %v\n", job.CinemaId, job.FilmId, err)
			continue
		}

		aggregateBookingWithResult(&result, booking)

		results <- result
		atomic.AddInt64(completed, 1)
		time.Sleep(time.Duration(options.RequestDelay) * time.Millisecond)
	}
}

// Function to fetch seats for all sessions across all showing groups
func fetchAllSeats(cinemaId string, showingResult *entities.ShowingResult, cookiesManager *header.CookiesManager) (map[string]*entities.Response, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	results := make(map[string]*entities.Response)
	errChan := make(chan error, 1)

	// Loop through all showing groups
	for _, group := range showingResult.ShowingGroups {
		// Loop through all sessions in each group
		for _, session := range group.Sessions {
			sessionId := session.SessionId
			wg.Add(1)

			// Use goroutine for concurrent requests
			go func(sessionId string) {
				defer wg.Done()

				// Create URL for this specific session
				url := fmt.Sprintf(constant.SEATS_URL,
					cinemaId, sessionId)
				client := &http.Client{}
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error creating request for session %s: %v", sessionId, err):
					default:
					}
					return // Just return without values
				}

				headers, err := header.GetHeaders(cookiesManager)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error getting headers for session %s: %v", sessionId, err):
					default:
					}
					return
				}
				for k, v := range headers {
					req.Header.Add(k, v)
				}

				resp, err := client.Do(req)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error making request for session %s: %v", sessionId, err):
					default:
					}
					return // Just return without values
				}
				defer resp.Body.Close()

				// Read and parse response
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error reading response for session %s: %v", sessionId, err):
					default:
					}
					return
				}

				var seatResponse entities.Response
				if err := json.Unmarshal(body, &seatResponse); err != nil {
					select {
					case errChan <- fmt.Errorf("error unmarshaling response for session %s: %v", sessionId, err):
					default:
					}
					return
				}

				// Store the result safely with mutex
				mutex.Lock()
				results[sessionId] = &seatResponse
				mutex.Unlock()
			}(sessionId)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if any errors occurred
	select {
	case err := <-errChan:
		return results, err
	default:
		return results, nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fetchShowing(cinemaId string, filmId string, regionData []entities.Region, cookiesManager *header.CookiesManager, options *FetchShowingsOptions) (entities.ShowingResult, error) {
	url := fmt.Sprintf(options.ShowingUrl, cinemaId, filmId)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return entities.ShowingResult{}, err
	}

	// Add headers from original code
	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return entities.ShowingResult{}, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return entities.ShowingResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return entities.ShowingResult{}, err
	}

	var showingResp entities.ShowingResponse
	if err := json.Unmarshal(body, &showingResp); err != nil {
		return entities.ShowingResult{}, err
	}

	// Check if results are empty
	if len(showingResp.Result) == 0 {
		return entities.ShowingResult{}, fmt.Errorf("no results found")
	}

	if len(showingResp.Result[0].ShowingGroups) == 0 {
		return entities.ShowingResult{}, nil
	}

	result := entities.ShowingResult{
		Movie:         showingResp.Result[0].FilmTitle,
		FilmId:        showingResp.Result[0].FilmId,
		CinemaId:      cinemaId,
		CinemaName:    getCinemaName(cinemaId, regionData),
		ShowingGroups: showingResp.Result[0].ShowingGroups,
	}

	return result, nil
}

func getCinemaName(cinemaId string, regionData []entities.Region) string {
	for _, reg := range regionData {
		for _, cinema := range reg.Cinemas {
			if cinema.CinemaId == cinemaId {
				return cinema.CinemaName
			}
		}
	}
	return ""
}

func aggregateBookingWithResult(result *entities.ShowingResult, booking map[string]*entities.Response) {
	for _, group := range result.ShowingGroups {
		for i := range group.Sessions {
			seatsNum := booking[group.Sessions[i].SessionId].Result.SeatRows.CountSeats()
			group.Sessions[i].Seats = int(booking[group.Sessions[i].SessionId].Result.SessionOccupancy * float64(seatsNum))
			group.Sessions[i].TotalSeats = seatsNum

			// Extract hour and minute from StartTime (format: 'YYYY-MM-DDTHH:MM:SS')
			if group.Sessions[i].StartTime != "" {
				parts := strings.Split(group.Sessions[i].StartTime, "T")
				if len(parts) == 2 {
					timePart := parts[1]
					timeParts := strings.Split(timePart, ":")
					if len(timeParts) >= 2 {
						hour := timeParts[0]
						minute := timeParts[1]
						group.Sessions[i].StartHour = hour + ":" + minute
						group.Sessions[i].RoundedStartHour = hour
					}
				}
			}
		}
	}
}
