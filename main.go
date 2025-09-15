package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"math/rand"

	"github.com/paologalligit/go-extractor/cookies"
)

// Cinema structs
type Cinema struct {
	CinemaId   string `json:"cinemaId"`
	CinemaName string `json:"cinemaName"`
}

type Region struct {
	Cinemas []Cinema `json:"cinemas"`
}

type CinemasFile struct {
	Result []Region `json:"result"`
}

// Film structs
type Film struct {
	FilmId string `json:"filmId"`
}

type FilmsFile struct {
	Result []Film `json:"result"`
}

// Showing structs
type Session struct {
	SessionId        string `json:"sessionId"`
	StartHour        string `json:"startHour"`
	RoundedStartHour string `json:"roundedStartHour"`
	EndHour          string `json:"endHour"`
	Seats            int    `json:"seats"`
	TotalSeats       int    `json:"totalSeats"`
	StartTime        string `json:"startTime"` // Added for extracting hour
}

type ShowingGroup struct {
	Date     string    `json:"date"`
	Sessions []Session `json:"sessions"`
}

type ShowingResponse struct {
	Result []struct {
		FilmId        string         `json:"filmId"`
		FilmTitle     string         `json:"filmTitle"`
		ShowingGroups []ShowingGroup `json:"showingGroups"`
	} `json:"result"`
}

type ShowingResult struct {
	Movie         string         `json:"movie"`
	FilmId        string         `json:"filmId"`
	CinemaId      string         `json:"cinemaId"`
	CinemaName    string         `json:"cinemaName"`
	ShowingGroups []ShowingGroup `json:"showingGroups"`
}

// BookingResult represents the top-level structure
type BookingResult struct {
	Result Result `json:"result"`
}

// Result contains the main seating information
type Result struct {
	SeatRows         []SeatRow `json:"seatRows"`
	SessionOccupancy float64   `json:"sessionOccupancy"`
}

// SeatRow represents a row of seats in the cinema
// Columns can contain nil for non-existent seats
type SeatRow struct {
	Columns []*Seat `json:"columns"`
}

// Seat represents an individual seat in the cinema
type Seat struct {
	SeatStatus int `json:"seatStatus,omitempty"`
}

// WorkItem represents a combination of cinema and film IDs to process
type WorkItem struct {
	CinemaId string
	FilmId   string
}

// Response struct for seat data
type Response struct {
	Result Result `json:"result"`
}

type ScheduledSession struct {
	Session    Session
	CinemaId   string
	CinemaName string
	FilmId     string
	FilmName   string
}

const (
	SHOWINGS_URL = "https://www.thespacecinema.it/api/microservice/showings/cinemas/%s/films?filmId=%s"
	SEATS_URL    = "https://www.thespacecinema.it/api/microservice/booking/Session/%s/%s/seats"
	CINEMAS_URL  = "https://www.thespacecinema.it/api/microservice/showings/cinemas"
	FILMS_URL    = "https://www.thespacecinema.it/api/microservice/showings/films"
)

func main() {

	// Parse command line flags
	maxGoroutines := flag.Int("workers", 10, "Number of concurrent workers")
	requestDelay := flag.Int("delay", 100, "Delay between requests in milliseconds")
	flag.Parse()

	fmt.Printf("Configuration: Using %d workers with %dms delay between requests\n", *maxGoroutines, *requestDelay)
	cookiesManager, err := cookies.New()
	if err != nil {
		panic(fmt.Sprintf("Failed to get cookies: %v", err))
	}
	fmt.Println("🍪 Cookies fetched")
	/*
		fetchCinemas(cookiesManager)
		fmt.Println("🏠 Cinemas fetched")
		fetchFilms(cookiesManager)
		fmt.Println("🎬 Films fetched")

		// Read cinema and film data
		cinemaIds, regionData := getCinemaIds()
		filmIds := getFilmIds()

		// Create work items (combinations of cinema and film IDs)
		var workItems []WorkItem
		for _, cinemaId := range cinemaIds {
			for _, filmId := range filmIds {
				workItems = append(workItems, WorkItem{CinemaId: cinemaId, FilmId: filmId})
			}
		}

		totalRequests := len(workItems)
		fmt.Printf("Total requests to make: %d\n", totalRequests)

		// Create channels for work distribution and result collection
		jobs := make(chan WorkItem, totalRequests)
		results := make(chan ShowingResult, totalRequests)

		// Counter for completed requests
		var completed int64 = 0

		// Start progress reporting in a separate goroutine
		stopProgress := make(chan struct{})
		go reportProgress(&completed, int64(totalRequests), stopProgress)

		// Start worker goroutines
		var wg sync.WaitGroup

		// Limit to maxGoroutines or total work items, whichever is smaller
		workerCount := min(*maxGoroutines, totalRequests)
		fmt.Printf("👷 Starting %d workers\n", workerCount)

		for range workerCount {
			wg.Add(1)
			go worker(jobs, results, &wg, regionData, *requestDelay, &completed, cookiesManager)
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
		var finalResults []ShowingResult
		for result := range results {
			finalResults = append(finalResults, result)
		}

		// Write results to file
		writeResultsToFile(finalResults, "showings")
		fmt.Println("\n🏁 Done! Results written to showings.json")
	*/
	const showingsFile = "showings_20250915_130905.json"
	fmt.Println("📂 Loading showings from", showingsFile)
	data, err := os.ReadFile(showingsFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to read %s: %v", showingsFile, err))
	}
	var finalResults []ShowingResult
	if err := json.Unmarshal(data, &finalResults); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal %s: %v", showingsFile, err))
	}
	fmt.Printf("📊 Loaded %d showings\n", len(finalResults))

	todaySessions := filterTodaySessions(finalResults)
	fmt.Println("🔍 Found", len(todaySessions), "sessions to schedule timers for")
	scheduleSessionTimers(todaySessions, cookiesManager)
}

func filterTodaySessions(results []ShowingResult) []ScheduledSession {
	todaySessions := []ScheduledSession{}
	today := time.Now().Format("2006-01-02")
	fmt.Println("🔍 Filtering today's sessions for date", today)
	for _, result := range results {
		for _, group := range result.ShowingGroups {
			groupTime, err := time.Parse("2006-01-02T15:04:05", group.Date)
			if err != nil {
				fmt.Printf("⚠️  Could not parse group date %s: %v\n", group.Date, err)
				continue
			}
			groupDate := groupTime.Format("2006-01-02")
			if groupDate == today {
				for _, session := range group.Sessions {
					todaySessions = append(todaySessions, ScheduledSession{
						Session:    session,
						CinemaId:   result.CinemaId,
						CinemaName: result.CinemaName,
						FilmId:     result.FilmId,
						FilmName:   result.Movie,
					})
				}
			}
		}
	}
	return todaySessions
}

func scheduleSessionTimers(todaySessions []ScheduledSession, cookiesManager *cookies.CookiesManager) {
	var wg sync.WaitGroup
	for _, session := range todaySessions {
		// Parse today's date and session's StartHour ("HH:MM") in local timezone
		loc := time.Now().Location()
		startTimeStr := time.Now().Format("2006-01-02") + "T" + session.Session.StartHour + ":00"
		startTime, err := time.ParseInLocation("2006-01-02T15:04:05", startTimeStr, loc)
		if err != nil {
			fmt.Printf("Failed to parse start time for session %s: %v\n", session.Session.SessionId, err)
			continue
		}
		targetTime := startTime.Add(13 * time.Minute)
		// Random delay between 100ms and 2m (120s)
		minDelay := 100 * time.Millisecond
		maxDelay := 2 * time.Minute
		deltaMillis := rand.Int63n(maxDelay.Milliseconds()-minDelay.Milliseconds()+1) + minDelay.Milliseconds()
		deltaDelay := time.Duration(deltaMillis) * time.Millisecond
		targetTime = targetTime.Add(deltaDelay)
		duration := time.Until(targetTime)
		if duration <= 0 {
			fmt.Printf("Session %s already started + 15min, skipping timer\n", session.Session.SessionId)
			continue
		}
		fmt.Printf("Scheduling timer for session %s with random delay %v (fires at %s)\n", session.Session.SessionId, deltaDelay, targetTime.Format(time.RFC3339))
		wg.Add(1)
		go func(s ScheduledSession, delay time.Duration) {
			fmt.Println("Im starting a timer with delay", delay, "for session starting at", s.Session.StartHour)
			defer wg.Done()
			timer := time.NewTimer(delay)
			<-timer.C
			fmt.Printf("Timer expired for session %s at %s, counting seats...\n", s.Session.SessionId, time.Now().Format(time.RFC3339))
			seatsNum, err := countSeatsForSession(s.Session, s.CinemaId, s.FilmId, cookiesManager)
			if err != nil {
				fmt.Printf("❌❌ Error counting seats for session %s: %v\n", s.Session.SessionId, err)
				return
			}
			if err := logSeatCount(s.CinemaName, s.FilmName, s.Session.SessionId, s.Session.StartHour, seatsNum); err != nil {
				fmt.Printf("❌❌ Error logging seat count for session %s: %v\n", s.Session.SessionId, err)
				return
			}
		}(session, duration)
	}
	wg.Wait()
}

func countSeatsForSession(session Session, cinemaId string, filmId string, cookiesManager *cookies.CookiesManager) (int, error) {
	fmt.Printf("Counting seats for session %s (cinema: %s, film: %s)\n", session.SessionId, cinemaId, filmId)
	url := fmt.Sprintf(SEATS_URL, cinemaId, session.SessionId)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request for session %s: %v", session.SessionId, err)
	}
	headers, err := getHeaders(cookiesManager)
	if err != nil {
		return 0, fmt.Errorf("error getting headers for session %s: %v", session.SessionId, err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request for session %s: %v", session.SessionId, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response for session %s: %v", session.SessionId, err)
	}
	var seatResponse Response
	if err := json.Unmarshal(body, &seatResponse); err != nil {
		return 0, fmt.Errorf("error unmarshaling response for session %s: %v", session.SessionId, err)
	}
	totalSeatsNumber := countSeats(seatResponse.Result.SeatRows)
	seatsNum := int(seatResponse.Result.SessionOccupancy * float64(totalSeatsNumber))
	fmt.Printf("Session %s: counted %d seats at %s\n", session.SessionId, seatsNum, time.Now().Format(time.RFC3339))
	return seatsNum, nil
}

func logSeatCount(cinemaName, filmName, sessionId, startHour string, seatsNum int) error {
	logEntry := SeatLogEntry{
		CinemaName: cinemaName,
		FilmName:   filmName,
		SessionId:  sessionId,
		Seats:      seatsNum,
		StartHour:  startHour,
		LoggedAt:   time.Now(),
	}
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	f, err := os.OpenFile("seat_counts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(logEntry); err != nil {
		return fmt.Errorf("error writing log entry: %v", err)
	}
	return nil
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

func worker(jobs <-chan WorkItem, results chan<- ShowingResult, wg *sync.WaitGroup, regionData []Region, requestDelay int, completed *int64, cookiesManager *cookies.CookiesManager) {
	defer wg.Done()

	for job := range jobs {
		result, err := fetchShowing(job.CinemaId, job.FilmId, regionData, cookiesManager)
		if err != nil {
			fmt.Printf("\nError fetching showing for cinema %s, film %s: %v\n", job.CinemaId, job.FilmId, err)
			continue
		}
		if result.FilmId == "" {
			continue
		}

		booking, err := fetchAllSeats(job.CinemaId, &result, cookiesManager)
		if err != nil {
			fmt.Printf("\nError fetching booking for cinema %s, film %s: %v\n", job.CinemaId, job.FilmId, err)
			continue
		}

		aggregateBookingWithResult(&result, booking)

		results <- result
		atomic.AddInt64(completed, 1)
		time.Sleep(time.Duration(requestDelay) * time.Millisecond)
	}
}

func aggregateBookingWithResult(result *ShowingResult, booking map[string]*Response) {
	for _, group := range result.ShowingGroups {
		for i := range group.Sessions {
			seatsNum := countSeats(booking[group.Sessions[i].SessionId].Result.SeatRows)
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

func countSeats(seatRows []SeatRow) int {
	total := 0
	for _, seatRow := range seatRows {
		for _, seat := range seatRow.Columns {
			if seat != nil {
				total++
			}
		}
	}
	return total
}

func fetchShowing(cinemaId string, filmId string, regionData []Region, cookiesManager *cookies.CookiesManager) (ShowingResult, error) {
	url := fmt.Sprintf(SHOWINGS_URL, cinemaId, filmId)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ShowingResult{}, err
	}

	// Add headers from original code
	headers, err := getHeaders(cookiesManager)
	if err != nil {
		return ShowingResult{}, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ShowingResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ShowingResult{}, err
	}

	var showingResp ShowingResponse
	if err := json.Unmarshal(body, &showingResp); err != nil {
		return ShowingResult{}, err
	}

	// Check if results are empty
	if len(showingResp.Result) == 0 {
		return ShowingResult{}, fmt.Errorf("no results found")
	}

	if len(showingResp.Result[0].ShowingGroups) == 0 {
		return ShowingResult{}, nil
	}

	result := ShowingResult{
		Movie:         showingResp.Result[0].FilmTitle,
		FilmId:        showingResp.Result[0].FilmId,
		CinemaId:      cinemaId,
		CinemaName:    getCinemaName(cinemaId, regionData),
		ShowingGroups: showingResp.Result[0].ShowingGroups,
	}

	return result, nil
}

func getCinemaIds() ([]string, []Region) {
	file, err := os.Open("cinemas.json")
	if err != nil {
		panic(fmt.Sprintf("Failed to open cinemas.json: %v", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Sprintf("Failed to read cinemas.json: %v", err))
	}

	var cinemas CinemasFile
	if err := json.Unmarshal(data, &cinemas); err != nil {
		panic(fmt.Sprintf("Failed to parse cinemas.json: %v", err))
	}

	var cinemaIds []string
	for _, reg := range cinemas.Result {
		for _, cinema := range reg.Cinemas {
			cinemaIds = append(cinemaIds, cinema.CinemaId)
		}
	}

	return cinemaIds, cinemas.Result
}

func getFilmIds() []string {
	file, err := os.Open("films.json")
	if err != nil {
		panic(fmt.Sprintf("Failed to open films.json: %v", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Sprintf("Failed to read films.json: %v", err))
	}

	var films FilmsFile
	if err := json.Unmarshal(data, &films); err != nil {
		panic(fmt.Sprintf("Failed to parse films.json: %v", err))
	}

	var filmIds []string
	for _, film := range films.Result {
		filmIds = append(filmIds, film.FilmId)
	}

	return filmIds
}

func getCinemaName(cinemaId string, regionData []Region) string {
	for _, reg := range regionData {
		for _, cinema := range reg.Cinemas {
			if cinema.CinemaId == cinemaId {
				return cinema.CinemaName
			}
		}
	}
	return ""
}

func writeResultsToFile(results []ShowingResult, baseFilename string) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.json", baseFilename, timestamp)
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal results: %v", err))
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write results to file: %v", err))
	}
	fmt.Printf("\nDone! Results written to %s\n", filename)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Function to fetch seats for all sessions across all showing groups
func fetchAllSeats(cinemaId string, showingResult *ShowingResult, cookiesManager *cookies.CookiesManager) (map[string]*Response, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	results := make(map[string]*Response)
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
				url := fmt.Sprintf(SEATS_URL,
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

				headers, err := getHeaders(cookiesManager)
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

				var seatResponse Response
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

func getHeaders(cookiesManager *cookies.CookiesManager) (map[string]string, error) {
	cookies, err := cookiesManager.GetCookies()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"cookie": cookies,
	}, nil
}

func fetchCinemas(cookiesManager *cookies.CookiesManager) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", CINEMAS_URL, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create request for cinemas: %v", err))
	}

	headers, err := getHeaders(cookiesManager)
	if err != nil {
		panic(fmt.Sprintf("Failed to get headers: %v", err))
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to fetch cinemas: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response for cinemas: %v", err))
	}

	var cinemas CinemasFile
	if err := json.Unmarshal(body, &cinemas); err != nil {
		panic(fmt.Sprintf("Failed to parse cinemas: %v", err))
	}

	os.WriteFile("cinemas.json", body, 0644)
}

func fetchFilms(cookiesManager *cookies.CookiesManager) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", FILMS_URL, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create request for films: %v", err))
	}

	headers, err := getHeaders(cookiesManager)
	if err != nil {
		panic(fmt.Sprintf("Failed to get headers: %v", err))
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to fetch films: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response for films: %v", err))
	}

	var films FilmsFile
	if err := json.Unmarshal(body, &films); err != nil {
		panic(fmt.Sprintf("Failed to parse films: %v", err))
	}

	os.WriteFile("films.json", body, 0644)
}

var logFileMutex sync.Mutex

type SeatLogEntry struct {
	CinemaName string    `json:"cinemaName"`
	FilmName   string    `json:"filmName"`
	SessionId  string    `json:"sessionId"`
	Seats      int       `json:"seats"`
	LoggedAt   time.Time `json:"loggedAt"`
	StartHour  string    `json:"startHour"`
}
