package fetchshowings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/team"
	"github.com/paologalligit/go-extractor/utils"
)

type FetchShowingsOptions struct {
	MaxGoroutines  int
	RequestDelay   int
	ShowingUrl     string
	OutputFileName string
	Client         client.Extractor
}

// RunFetchShowings fetches showings and writes them to a file
func RunFetchShowings(options *FetchShowingsOptions) error {
	client := options.Client
	cinemaFile, err := client.GetCinemas()
	if err != nil {
		return fmt.Errorf("failed to fetch cinemas: %w", err)
	}
	data, err := json.MarshalIndent(cinemaFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cinemas: %w", err)
	}
	os.WriteFile(filepath.Join(constant.FilesPath, "cinemas.json"), data, 0644)
	fmt.Println("🏠 Cinemas fetched")

	filmFile, err := client.GetFilms()
	if err != nil {
		return fmt.Errorf("failed to fetch films: %w", err)
	}
	data, err = json.MarshalIndent(filmFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal films: %w", err)
	}
	os.WriteFile(filepath.Join(constant.FilesPath, "films.json"), data, 0644)
	fmt.Println("🎬 Films fetched")

	// Read cinema and film data
	cinemaIds, regionData, err := utils.GetCinemaIds()
	if err != nil {
		return fmt.Errorf("failed to get cinema ids: %w", err)
	}
	filmIds, err := utils.GetFilmIds()
	if err != nil {
		return fmt.Errorf("failed to get film ids: %w", err)
	}

	// Create work items (combinations of cinema and film IDs)
	var workItems []entities.WorkItem
	for _, cinemaId := range cinemaIds {
		for _, filmId := range filmIds {
			workItems = append(workItems, entities.WorkItem{CinemaId: cinemaId, FilmId: filmId})
		}
	}
	totalRequests := len(workItems)
	fmt.Printf("Total requests to make: %d\n", totalRequests)

	// Progress reporting
	var completed int64 = 0
	stopProgress := make(chan struct{})
	go utils.ReportProgress(&completed, int64(totalRequests), stopProgress)

	// Use the team package for the worker pool
	workerCount := min(options.MaxGoroutines, totalRequests)
	fmt.Printf("👷 Starting %d workers\n", workerCount)

	fetchTeam := team.NewFetchTeam(workerCount, &team.FetchTeamWorkingMaterial{
		Client:       options.Client,
		ShowingUrl:   options.ShowingUrl,
		RequestDelay: options.RequestDelay,
		RegionData:   regionData,
	})
	finalResults := fetchTeam.Run(workItems)

	// Write results to file
	if err := utils.WriteResultsToFile(finalResults, options.OutputFileName); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	fmt.Println("\n🏁 Done! Results written to", options.OutputFileName, ".json")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
