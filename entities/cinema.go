package entities

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

type Platea []SeatRow

type Result struct {
	SeatRows         Platea  `json:"seatRows"`
	SessionOccupancy float64 `json:"sessionOccupancy"`
}

type Seat struct {
	SeatStatus int `json:"seatStatus,omitempty"`
}

type SeatRow struct {
	Columns []*Seat `json:"columns"`
}

func (p *Platea) CountSeats() int {
	total := 0
	for _, seatRow := range *p {
		total += seatRow.countSeats()
	}
	return total
}

func (s *SeatRow) countSeats() int {
	total := 0
	for _, seat := range s.Columns {
		if seat != nil {
			total++
		}
	}
	return total
}
