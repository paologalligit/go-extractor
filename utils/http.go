package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
)

func GetCinemaIds() ([]string, []entities.Region, error) {
	file, err := os.Open(filepath.Join(constant.FilesPath, "cinemas.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cinemas.json: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read cinemas.json: %w", err)
	}

	var cinemas entities.CinemasFile
	if err = json.Unmarshal(data, &cinemas); err != nil {
		return nil, nil, fmt.Errorf("failed to parse cinemas.json: %w", err)
	}

	var cinemaIds []string
	for _, reg := range cinemas.Result {
		for _, cinema := range reg.Cinemas {
			cinemaIds = append(cinemaIds, cinema.CinemaId)
		}
	}

	return cinemaIds, cinemas.Result, nil
}

func GetFilmIds() ([]string, error) {
	file, err := os.Open(filepath.Join(constant.FilesPath, "films.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to open films.json: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read films.json: %w", err)
	}

	var films entities.FilmsFile
	if err := json.Unmarshal(data, &films); err != nil {
		return nil, fmt.Errorf("failed to parse films.json: %w", err)
	}

	var filmIds []string
	for _, film := range films.Result {
		filmIds = append(filmIds, film.FilmId)
	}

	return filmIds, nil
}
