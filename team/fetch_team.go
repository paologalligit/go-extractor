package team

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/utils"
)

type FetchTeamWorkingMaterial struct {
	RequestDelay int
	RegionData   []entities.Region
	ShowingUrl   string
	Completed    *int64
	Client       client.Extractor
}

type FetchTeam struct {
	WorkerCount     int
	WorkingMaterial *FetchTeamWorkingMaterial
}

func NewFetchTeam(workerCount int, wm *FetchTeamWorkingMaterial) *FetchTeam {
	return &FetchTeam{
		WorkerCount:     workerCount,
		WorkingMaterial: wm,
	}
}

func (ft *FetchTeam) Run(workItems []entities.WorkItem) []entities.ShowingResult {
	// Stage 1: Fetch showings for each work item
	showingTeam := Team[entities.WorkItem, entities.ShowingResult]{
		WorkerCount: ft.WorkerCount,
		Worker: func(job entities.WorkItem) (entities.ShowingResult, error) {
			result, err := ft.fetchShowing(job.CinemaId, job.FilmId, ft.WorkingMaterial.RegionData, ft.WorkingMaterial.ShowingUrl)
			if err != nil {
				return entities.ShowingResult{}, fmt.Errorf("error fetching showing for cinema %s, film %s: %w", job.CinemaId, job.FilmId, err)
			}
			if result.FilmId == "" {
				return entities.ShowingResult{}, nil
			}
			return result, nil
		},
	}
	showingResults := showingTeam.Run(workItems)

	var filteredShowings []entities.ShowingResult
	for _, result := range showingResults {
		if result.FilmId != "" {
			filteredShowings = append(filteredShowings, result)
		}
	}

	// Stage 2: For each showing, fetch all seats and aggregate
	seatsTeam := Team[entities.ShowingResult, entities.ShowingResult]{
		WorkerCount: ft.WorkerCount,
		Worker: func(showing entities.ShowingResult) (entities.ShowingResult, error) {
			booking, err := ft.fetchAllSeats(showing.CinemaId, &showing)
			if err != nil {
				return entities.ShowingResult{}, fmt.Errorf("error fetching booking for cinema %s, film %s: %w", showing.CinemaId, showing.FilmId, err)
			}
			aggregateBookingWithResult(&showing, booking)
			if ft.WorkingMaterial.Completed != nil {
				atomic.AddInt64(ft.WorkingMaterial.Completed, 1)
			}
			time.Sleep(time.Duration(ft.WorkingMaterial.RequestDelay) * time.Millisecond)
			return showing, nil
		},
	}
	finalResults := seatsTeam.Run(filteredShowings)
	return finalResults
}

func (ft *FetchTeam) fetchShowing(cinemaId string, filmId string, regionData []entities.Region, showingUrl string) (entities.ShowingResult, error) {
	url := fmt.Sprintf(showingUrl, cinemaId, filmId)
	showingResp, err := ft.WorkingMaterial.Client.CallShowings(url)
	if err != nil {
		return entities.ShowingResult{}, err
	}
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
		CinemaName:    utils.GetCinemaName(cinemaId, regionData),
		ShowingGroups: showingResp.Result[0].ShowingGroups,
	}
	return result, nil
}

func (ft *FetchTeam) fetchAllSeats(cinemaId string, showingResult *entities.ShowingResult) (map[string]*entities.Response, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	results := make(map[string]*entities.Response)
	errChan := make(chan error, 1)

	for _, group := range showingResult.ShowingGroups {
		for _, session := range group.Sessions {
			sessionId := session.SessionId
			wg.Add(1)
			go func(sessionId string) {
				defer wg.Done()
				url := fmt.Sprintf(constant.SEATS_URL, cinemaId, sessionId)
				seatResponse, err := ft.WorkingMaterial.Client.CallSeats(url)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("error making request for session %s: %v", sessionId, err):
					default:
					}
					return
				}
				mutex.Lock()
				results[sessionId] = seatResponse
				mutex.Unlock()
			}(sessionId)
		}
	}
	wg.Wait()
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
