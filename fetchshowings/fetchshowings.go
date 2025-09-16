package fetchshowings

import (
	"fmt"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/team"
	"github.com/paologalligit/go-extractor/utils"
)

type FetchShowingsOptions struct {
	MaxGoroutines  int
	RequestDelay   int
	ShowingUrl     string
	OutputFileName string
	CookiesManager *header.CookiesManager
}

// RunFetchShowings fetches showings and writes them to a file
func RunFetchShowings(options *FetchShowingsOptions) error {
	if err := utils.FetchCinemas(options.CookiesManager); err != nil {
		return fmt.Errorf("failed to fetch cinemas: %w", err)
	}
	fmt.Println("🏠 Cinemas fetched")
	if err := utils.FetchFilms(options.CookiesManager); err != nil {
		return fmt.Errorf("failed to fetch films: %w", err)
	}
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
		Client:       client.New(options.CookiesManager),
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
