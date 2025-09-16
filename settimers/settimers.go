package settimers

import (
	"context"
	"fmt"
	"time"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/persistence"
	"github.com/paologalligit/go-extractor/team"
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
	cinemaIds, regionData, err := utils.GetCinemaIds()
	if err != nil {
		return fmt.Errorf("error getting cinema ids: %w", err)
	}

	wm := &team.SessionTeamWorkingMaterial{
		RequestDelay:  options.RequestDelay,
		Client:        client.New(options.CookiesManager),
		MaxGoroutines: options.MaxGoroutines,
		CinemaIds:     cinemaIds,
		RegionData:    regionData,
	}

	st := team.NewSessionTeam(options.MaxGoroutines, wm)
	st.Run(today, todayFile, func(s entities.ScheduledSession) {
		// This callback is executed when the timer fires for a session
		showingUrl := constant.SHOWINGS_URL_TODAY + today + constant.SHOWINGS_URL_TODAY_PARAMS
		url := fmt.Sprintf(showingUrl, s.CinemaId, s.Session.SessionId)
		seatResp, err := st.WorkingMaterial.Client.CallSeats(url)
		if err != nil {
			fmt.Printf("❌❌ Error counting seats for session %s: %v\n", s.Session.SessionId, err)
			return
		}
		totalSeats := seatResp.Result.SeatRows.CountSeats()
		seatsNum := int(seatResp.Result.SessionOccupancy * float64(totalSeats))
		entry := entities.SeatLogEntry{
			CinemaName: s.CinemaName,
			FilmName:   s.FilmName,
			SessionId:  s.Session.SessionId,
			Seats:      seatsNum,
			StartHour:  s.Session.StartHour,
			LoggedAt:   time.Now(),
		}
		if err := options.Persistence.WriteSessionSeats(context.Background(), entry); err != nil {
			fmt.Printf("❌❌ Error logging seat count for session %s: %v\n", s.Session.SessionId, err)
		}
	})
	return nil
}
