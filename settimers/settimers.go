package settimers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/fetchshowings"
	"github.com/paologalligit/go-extractor/header"
)

type SettimersOptions struct {
	CookiesManager *header.CookiesManager
}

func RunSeatTimers(options *SettimersOptions) error {
	today := time.Now().Format("2006-01-02")
	todayFile := fmt.Sprintf("todaySession-%s.json", today)

	if err := upsertTodayFile(today, todayFile, options.CookiesManager); err != nil {
		return fmt.Errorf("error checking for today's sessions: %w", err)
	}
	fmt.Println("[settimers] Would now read and schedule seat timers.")

	todaySessions, err := readTodaySessions(todayFile)
	if err != nil {
		return fmt.Errorf("error reading today's sessions: %w", err)
	}

	scheduleSessionTimers(todaySessions, options.CookiesManager)
	return nil
}

func scheduleSessionTimers(todaySessions []entities.ScheduledSession, cookiesManager *header.CookiesManager) {
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
			if err := logSeatCount(s.CinemaName, s.FilmName, s.Session.SessionId, s.Session.StartHour, seatsNum); err != nil {
				fmt.Printf("❌❌ Error logging seat count for session %s: %v\n", s.Session.SessionId, err)
				return
			}
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

func logSeatCount(cinemaName, filmName, sessionId, startHour string, seatsNum int) error {
	logEntry := entities.SeatLogEntry{
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

var logFileMutex sync.Mutex

func upsertTodayFile(today, todayFile string, cookiesManager *header.CookiesManager) error {
	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		fmt.Printf("%s not found, fetching showings for today...\n", todayFile)
		showingsUrl := fmt.Sprintf("%s?showingDate=%sT00:00:00", constant.SHOWINGS_URL, today)
		opt := &fetchshowings.FetchShowingsOptions{
			MaxGoroutines:  10,
			RequestDelay:   100,
			ShowingUrl:     showingsUrl,
			OutputFileName: todayFile,
			CookiesManager: cookiesManager,
		}
		if err := fetchshowings.RunFetchShowings(opt); err != nil {
			return fmt.Errorf("error fetching showings: %w", err)
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
	var sessions []entities.ScheduledSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", todayFile, err)
	}
	return sessions, nil
}
