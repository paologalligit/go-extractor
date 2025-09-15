package entities

type Film struct {
	FilmId string `json:"filmId"`
}

type FilmsFile struct {
	Result []Film `json:"result"`
}
