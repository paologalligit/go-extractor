package team

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/paologalligit/go-extractor/entities"
	"github.com/stretchr/testify/assert"
)

const FILE_PATH_SESSIONS_TEST = "/Users/paologalli/Documents/go_extractor/team/today_sessions.json"

var writingMutex sync.Mutex

func TestSessionTeam_Run(t *testing.T) {
	// Arrange
	wm := &SessionTeamWorkingMaterial{
		RequestDelay:  10,
		Client:        &MockFetchExtractor{},
		MaxGoroutines: 2,
		Delay: func(d time.Duration) <-chan time.Time {
			ch := make(chan time.Time, 1)
			ch <- time.Now()
			return ch
		},
		CinemaIds: []string{"1030"},
		RegionData: []entities.Region{
			{
				Cinemas: []entities.Cinema{
					{CinemaId: "1030", CinemaName: "Vimercate"},
				},
			},
		},
	}
	st := NewSessionTeam(2, wm)

	today := "2025-09-15"
	todayFile := "./todaySession-2025-09-15.json"
	defer func() {
		os.Remove(todayFile)
	}()

	var called []entities.ScheduledSession
	callback := func(s entities.ScheduledSession) {
		writingMutex.Lock()
		called = append(called, s)
		writingMutex.Unlock()
	}

	// Act
	sessions, err := st.Run(today, todayFile, callback)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, sessions)
	assert.Equal(t, len(sessions), len(called))
}

type MockSessionExtractor struct{}

func (m *MockSessionExtractor) CallSeats(url string) (*entities.Response, error) {
	data, err := os.ReadFile(FILE_PATH_SEATS_TEST)
	if err != nil {
		return nil, err
	}
	var resp entities.Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *MockSessionExtractor) CallShowings(url string) (*entities.ShowingResponse, error) {
	data, err := os.ReadFile(FILE_PATH_SESSIONS_TEST)
	if err != nil {
		return nil, err
	}
	var resp entities.ShowingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
