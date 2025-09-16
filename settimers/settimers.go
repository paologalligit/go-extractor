package settimers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/persistence"
	"github.com/paologalligit/go-extractor/utils"
)

type SettimersOptions struct {
	CookiesManager *header.CookiesManager
	Persistence    persistence.Persistence
	MaxGoroutines  int
	RequestDelay   int
}

func RunSeatTimers(options *SettimersOptions) error {
	today := time.Now().Format("2006-01-02")
	todayFile := fmt.Sprintf("todaySession-%s.json", today)

	if err := upsertTodayFile(today, todayFile, options); err != nil {
		return fmt.Errorf("error checking for today's sessions: %w", err)
	}
	fmt.Println("Today file found or created, reading and scheduling seat timers.")

	todaySessions, err := readTodaySessions(todayFile)
	if err != nil {
		return fmt.Errorf("error reading today's sessions: %w", err)
	}

	scheduleSessionTimers(todaySessions, options.CookiesManager, options.Persistence)
	return nil
}

func scheduleSessionTimers(todaySessions []entities.ScheduledSession, cookiesManager *header.CookiesManager, persistence persistence.Persistence) {
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
		targetTime := startTime.Add(12 * time.Minute)
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
		go func(s entities.ScheduledSession, delay time.Duration) {
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
			if err := persistence.WriteSessionSeats(context.Background(), entities.SeatLogEntry{
				CinemaName: s.CinemaName,
				FilmName:   s.FilmName,
				SessionId:  s.Session.SessionId,
				Seats:      seatsNum,
				StartHour:  s.Session.StartHour,
				LoggedAt:   time.Now(),
			}); err != nil {
				fmt.Printf("❌❌ Error logging seat count for session %s: %v\n", s.Session.SessionId, err)
				return
			}
			fmt.Println("💾 Data stored on database correctly")
		}(session, duration)
	}
	wg.Wait()
}

func countSeatsForSession(session entities.Session, cinemaId string, filmId string, cookiesManager *header.CookiesManager) (int, error) {
	fmt.Printf("Counting seats for session %s (cinema: %s, film: %s)\n", session.SessionId, cinemaId, filmId)
	url := fmt.Sprintf(constant.SEATS_URL, cinemaId, session.SessionId)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request for session %s: %v", session.SessionId, err)
	}
	headers, err := header.GetHeaders(cookiesManager)
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
	var seatResponse entities.Response
	if err := json.Unmarshal(body, &seatResponse); err != nil {
		return 0, fmt.Errorf("error unmarshaling response for session %s: %v", session.SessionId, err)
	}
	totalSeatsNumber := seatResponse.Result.SeatRows.CountSeats()
	seatsNum := int(seatResponse.Result.SessionOccupancy * float64(totalSeatsNumber))
	fmt.Printf("Session %s: counted %d seats at %s\n", session.SessionId, seatsNum, time.Now().Format(time.RFC3339))
	return seatsNum, nil
}

func upsertTodayFile(today, todayFile string, options *SettimersOptions) error {
	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		fmt.Printf("%s not found, fetching showings for today...\n", todayFile)
		if err := fetchTodayShowings(today, todayFile, options); err != nil {
			return fmt.Errorf("error fetching today's showings: %w", err)
		}
		return nil
	}

	fmt.Printf("Found %s, using existing file.\n", todayFile)
	return nil
}

func readTodaySessions(todayFile string) ([]entities.ScheduledSession, error) {
	data, err := os.ReadFile(todayFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", todayFile, err)
	}
	var showingResults []entities.ShowingResult
	if err := json.Unmarshal(data, &showingResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", todayFile, err)
	}

	return convertToScheduledSessions(showingResults), nil
}

func fetchTodayShowings(today, todayFile string, options *SettimersOptions) error {
	cinemaIds, regionData, err := utils.GetCinemaIds()
	if err != nil {
		return fmt.Errorf("error getting cinema ids: %w", err)
	}

	totalRequests := len(cinemaIds)
	// Create channels for work distribution and result collection
	jobs := make(chan string, totalRequests)
	results := make(chan entities.ShowingResult, totalRequests)

	// Counter for completed requests
	var completed int64 = 0

	// Start progress reporting in a separate goroutine
	stopProgress := make(chan struct{})
	go utils.ReportProgress(&completed, int64(totalRequests), stopProgress)

	// Start worker goroutines
	var wg sync.WaitGroup

	// Limit to maxGoroutines or total work items, whichever is smaller
	workerCount := min(options.MaxGoroutines, totalRequests)
	fmt.Printf("👷 Starting %d workers\n", workerCount)

	for range workerCount {
		wg.Add(1)
		go worker(jobs, results, &wg, &completed, today, options, regionData)
	}

	// Add jobs to channel
	for _, item := range cinemaIds {
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
	if err := utils.WriteResultsToFile(finalResults, todayFile); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	fmt.Println("\n🏁 Done! Results written to", todayFile, ".json")
	return nil
}

func worker(jobs <-chan string, results chan<- entities.ShowingResult, wg *sync.WaitGroup, completed *int64, today string, options *SettimersOptions, regionData []entities.Region) {
	defer wg.Done()

	for job := range jobs {
		showings, err := fetchShowing(job, options.CookiesManager, today, regionData)
		if err != nil {
			fmt.Printf("\nError fetching showing for cinema %s: %v\n", job, err)
			continue
		}
		if len(showings) == 0 {
			continue
		}

		for _, result := range showings {
			booking, err := fetchAllSeats(job, &result, options.CookiesManager)
			if err != nil {
				fmt.Printf("\nError fetching booking for cinema %s: %v\n", job, err)
				continue
			}

			aggregateBookingWithResult(&result, booking)
			results <- result
			atomic.AddInt64(completed, 1)
			time.Sleep(time.Duration(options.RequestDelay) * time.Millisecond)
		}
	}
}

func fetchShowing(cinemaId string, cookiesManager *header.CookiesManager, today string, regionData []entities.Region) ([]entities.ShowingResult, error) {
	url := constant.SHOWINGS_URL_TODAY + today + constant.SHOWINGS_URL_TODAY_PARAMS
	url = fmt.Sprintf(url, cinemaId)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers from original code
	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var showingResp entities.ShowingResponse
	if err := json.Unmarshal(body, &showingResp); err != nil {
		return nil, err
	}

	// Check if results are empty
	if len(showingResp.Result) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	var results []entities.ShowingResult
	for _, result := range showingResp.Result {
		results = append(results, entities.ShowingResult{
			Movie:         result.FilmTitle,
			FilmId:        result.FilmId,
			CinemaId:      cinemaId,
			CinemaName:    utils.GetCinemaName(cinemaId, regionData),
			ShowingGroups: result.ShowingGroups,
		})
	}
	return results, nil
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
				url := fmt.Sprintf(constant.SEATS_URL, cinemaId, sessionId)
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

func convertToScheduledSessions(showingResults []entities.ShowingResult) []entities.ScheduledSession {
	var scheduledSessions []entities.ScheduledSession
	for _, showingResult := range showingResults {
		for _, group := range showingResult.ShowingGroups {
			for _, session := range group.Sessions {
				scheduledSessions = append(scheduledSessions, entities.ScheduledSession{
					Session:    session,
					CinemaId:   showingResult.CinemaId,
					CinemaName: showingResult.CinemaName,
					FilmId:     showingResult.FilmId,
					FilmName:   showingResult.Movie,
				})
			}
		}
	}
	return scheduledSessions
}
