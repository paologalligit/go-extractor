package entities

import (
	"time"
)

type Session struct {
	SessionId        string `json:"sessionId"`
	StartHour        string `json:"startHour"`
	RoundedStartHour string `json:"roundedStartHour"`
	Seats            int    `json:"seats"`
	TotalSeats       int    `json:"totalSeats"`
	StartTime        string `json:"startTime"`
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

type BookingResult struct {
	Result Result `json:"result"`
}
type SeatLogEntry struct {
	CinemaName string    `json:"cinemaName"`
	FilmName   string    `json:"filmName"`
	SessionId  string    `json:"sessionId"`
	Seats      int       `json:"seats"`
	LoggedAt   time.Time `json:"loggedAt"`
	StartHour  string    `json:"startHour"`
}

type ScheduledSession struct {
	Session    Session
	CinemaId   string
	CinemaName string
	FilmId     string
	FilmName   string
}

type WorkItem struct {
	CinemaId string
	FilmId   string
}

type Response struct {
	Result Result `json:"result"`
}
