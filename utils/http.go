package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
)

func FetchCinemas(cookiesManager *header.CookiesManager) error {
	if _, err := os.Stat(filepath.Join(constant.FilesPath, "cinemas.json")); err == nil {
		return nil
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", constant.CINEMAS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for cinemas: %w", err)
	}

	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return fmt.Errorf("failed to get headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch cinemas: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for cinemas: %w", err)
	}

	var cinemas entities.CinemasFile
	if err := json.Unmarshal(body, &cinemas); err != nil {
		return fmt.Errorf("failed to parse cinemas: %w", err)
	}

	os.WriteFile(filepath.Join(constant.FilesPath, "cinemas.json"), body, 0644)
	return nil
}

func FetchFilms(cookiesManager *header.CookiesManager) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", constant.FILMS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for films: %w", err)
	}

	headers, err := header.GetHeaders(cookiesManager)
	if err != nil {
		return fmt.Errorf("failed to get headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch films: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for films: %w", err)
	}

	var films entities.FilmsFile
	if err := json.Unmarshal(body, &films); err != nil {
		return fmt.Errorf("failed to parse films: %w", err)
	}

	os.WriteFile(filepath.Join(constant.FilesPath, "films.json"), body, 0644)
	return nil
}

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
