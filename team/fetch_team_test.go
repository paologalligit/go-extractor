package team

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/stretchr/testify/assert"
)

const (
	FILE_PATH_SHOWINGS_TEST = "/Users/paologalli/Documents/go_extractor/team/call_showings_template.json"
	FILE_PATH_SEATS_TEST    = "/Users/paologalli/Documents/go_extractor/team/seats_template.json"
)

func TestFetchTeam(t *testing.T) {
	// Arrange
	extractor := &MockFetchExtractor{}
	ftwm := &FetchTeamWorkingMaterial{
		Client:       extractor,
		ShowingUrl:   constant.SHOWINGS_URL,
		RequestDelay: 100,
		RegionData: []entities.Region{
			{
				Cinemas: []entities.Cinema{
					{
						CinemaId:   "1030",
						CinemaName: "Vimercate",
					},
				},
			},
		},
	}

	// Act
	ft := NewFetchTeam(1, ftwm)
	result := ft.Run([]entities.WorkItem{{CinemaId: "1030", FilmId: "HO00003077"}})

	// Assert
	assert.NotEmpty(t, result)
	assert.Equal(t, len(result), 1)
	session := result[0]
	assert.Equal(t, session.FilmId, "HO00003077")
	assert.Equal(t, session.CinemaId, "1030")
	assert.Equal(t, session.CinemaName, "Vimercate")
	assert.Equal(t, session.Movie, "The Conjuring: Il rito finale")
	assert.Equal(t, len(session.ShowingGroups), 8)
	for _, showing := range session.ShowingGroups {
		assert.NotEmpty(t, showing.Date)
		assert.NotEmpty(t, showing.Sessions)
		for _, session := range showing.Sessions {
			assert.NotEmpty(t, session.SessionId)
			assert.NotEmpty(t, session.StartHour)
			assert.Greater(t, session.Seats, 0)
			assert.Greater(t, session.TotalSeats, 0)
			assert.NotEmpty(t, session.StartTime)
			assert.NotEmpty(t, session.RoundedStartHour)
		}
	}
}

type MockFetchExtractor struct{}

func (m *MockFetchExtractor) CallShowings(url string) (*entities.ShowingResponse, error) {
	data, err := os.ReadFile(FILE_PATH_SHOWINGS_TEST)
	if err != nil {
		return nil, err
	}
	var resp entities.ShowingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *MockFetchExtractor) CallSeats(url string) (*entities.Response, error) {
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
