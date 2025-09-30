package team

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/utils"
)

type DelayFunc func(time.Duration) <-chan time.Time

type SessionTeamWorkingMaterial struct {
	RequestDelay  int
	Completed     *int64
	Client        client.Extractor
	MaxGoroutines int
	CinemaIds     []string
	RegionData    []entities.Region
	Delay         DelayFunc // Injected delay function for timers
}

type SessionTeam struct {
	WorkerCount     int
	WorkingMaterial *SessionTeamWorkingMaterial
}

func NewSessionTeam(workerCount int, wm *SessionTeamWorkingMaterial) *SessionTeam {
	return &SessionTeam{
		WorkerCount:     workerCount,
		WorkingMaterial: wm,
	}
}

// Pipeline: For each ScheduledSession, schedule a timer, fetch seat data, and aggregate
func (st *SessionTeam) Run(today, todayFile string, callback func(s entities.ScheduledSession)) ([]entities.ScheduledSession, error) {
	// TODO: do we really need to save the today file to disk?
	if err := st.upsertTodayFile(today, todayFile); err != nil {
		return nil, fmt.Errorf("error upserting today file: %w", err)
	}

	todaySessions, err := st.readTodaySessions(todayFile)
	if err != nil {
		return nil, fmt.Errorf("error reading today's sessions: %w", err)
	}

	st.scheduleSessionTimers(todaySessions, callback)
	return todaySessions, nil
}

// upsertTodayFile checks for the today file and fetches showings if missing
func (st *SessionTeam) upsertTodayFile(today, todayFile string) error {
	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		fmt.Printf("%s not found, fetching showings for today...\n", todayFile)
		data, err := st.fetchTodayShowings(today)
		if err != nil {
			return fmt.Errorf("error fetching today's showings: %w", err)
		}

		if err := os.WriteFile(todayFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write results to file: %w", err)
		}
		fmt.Println("\n🏁 Done! Results written to", todayFile)
		return nil
	}
	fmt.Printf("Found %s, using existing file.\n", todayFile)
	return nil
}

// fetchTodayShowings fetches and writes today's showings to file
func (st *SessionTeam) fetchTodayShowings(today string) ([]byte, error) {
	totalRequests := len(st.WorkingMaterial.CinemaIds)
	workerCount := st.WorkingMaterial.MaxGoroutines
	if workerCount <= 0 || workerCount > totalRequests {
		workerCount = totalRequests
	}

	teamPool := Team[string, []entities.ShowingResult]{
		WorkerCount: workerCount,
		Worker: func(item string) ([]entities.ShowingResult, error) {
			url := fmt.Sprintf(constant.SHOWINGS_URL_TODAY+today+constant.SHOWINGS_URL_TODAY_PARAMS, item)
			showingResp, err := st.WorkingMaterial.Client.CallShowings(url)
			if err != nil {
				return nil, err
			}
			if len(showingResp.Result) == 0 {
				return nil, nil
			}
			var results []entities.ShowingResult
			for _, result := range showingResp.Result {
				showing := entities.ShowingResult{
					Movie:         result.FilmTitle,
					FilmId:        result.FilmId,
					CinemaId:      item,
					CinemaName:    utils.GetCinemaName(item, st.WorkingMaterial.RegionData),
					ShowingGroups: result.ShowingGroups,
				}
				// For each session, fetch seat data and set Seats/TotalSeats, StartHour, RoundedStartHour
				for gi := range showing.ShowingGroups {
					for si := range showing.ShowingGroups[gi].Sessions {
						session := &showing.ShowingGroups[gi].Sessions[si]
						seatUrl := fmt.Sprintf(constant.SEATS_URL, item, session.SessionId)
						seatResp, err := st.WorkingMaterial.Client.CallSeats(seatUrl)
						if err == nil && seatResp != nil {
							totalSeats := seatResp.Result.SeatRows.CountSeats()
							session.TotalSeats = totalSeats
							seatsNum := int(seatResp.Result.SessionOccupancy * float64(totalSeats))
							session.Seats = seatsNum
						}
						// Set StartHour and RoundedStartHour from StartTime
						if session.StartTime != "" {
							parts := strings.Split(session.StartTime, "T")
							if len(parts) == 2 {
								timePart := parts[1]
								timeParts := strings.Split(timePart, ":")
								if len(timeParts) >= 2 {
									hour := timeParts[0]
									minute := timeParts[1]
									session.StartHour = hour + ":" + minute
									session.RoundedStartHour = hour
								}
							}
						}
					}
				}
				results = append(results, showing)
			}
			return results, nil
		},
	}
	var allResults []entities.ShowingResult
	for _, results := range teamPool.Run(st.WorkingMaterial.CinemaIds) {
		allResults = append(allResults, results...)
	}
	return json.MarshalIndent(allResults, "", "  ")
}

func (st *SessionTeam) readTodaySessions(todayFile string) ([]entities.ScheduledSession, error) {
	data, err := os.ReadFile(todayFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", todayFile, err)
	}
	var showingResults []entities.ShowingResult
	if err := json.Unmarshal(data, &showingResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", todayFile, err)
	}

	for _, showingResult := range showingResults {
		aggregateWithHour(&showingResult)
	}

	return st.convertToScheduledSessions(showingResults), nil
}

func (st *SessionTeam) convertToScheduledSessions(showingResults []entities.ShowingResult) []entities.ScheduledSession {
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

// scheduleSessionTimers schedules a timer for each session and calls the provided callback when the timer fires.
func (st *SessionTeam) scheduleSessionTimers(sessions []entities.ScheduledSession, callback func(s entities.ScheduledSession)) {
	var wg sync.WaitGroup
	delayFunc := st.WorkingMaterial.Delay
	if delayFunc == nil {
		delayFunc = time.After
	}
	for _, session := range sessions {
		loc := time.Now().Location()
		now := time.Now()
		startTimeStr := now.Format("2006-01-02") + "T" + session.Session.StartHour + ":00"
		startTime, err := time.ParseInLocation("2006-01-02T15:04:05", startTimeStr, loc)
		if err == nil && startTime.Before(now) {
			// If the session's start time is before now, it's for tomorrow
			startTime = startTime.Add(24 * time.Hour)
		}
		if err != nil {
			fmt.Printf("Failed to parse start time for session %s: %v\n", session.Session.SessionId, err)
			continue
		}
		targetTime := startTime.Add(12 * time.Minute)
		if session.CinemaId == "1018" {
			targetTime = startTime.Add(-2 * time.Minute)
		}
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
			defer wg.Done()
			<-delayFunc(delay)
			fmt.Printf("Timer expired for session %s at %s, executing callback...\n", s.Session.SessionId, time.Now().Format(time.RFC3339))
			callback(s)
		}(session, duration)
	}
	wg.Wait()
}

func aggregateWithHour(result *entities.ShowingResult) {
	for _, group := range result.ShowingGroups {
		for i := range group.Sessions {
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
