package utils

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/paologalligit/go-extractor/entities"
)

func ReportProgress(completed *int64, total int64, stop chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			current := atomic.LoadInt64(completed)
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rProgress: %d/%d (%.2f%%) completed", current, total, percent)
		case <-stop:
			// Final progress update
			current := atomic.LoadInt64(completed)
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rProgress: %d/%d (%.2f%%) completed", current, total, percent)
			return
		}
	}
}

func GetCinemaName(cinemaId string, regionData []entities.Region) string {
	for _, reg := range regionData {
		for _, cinema := range reg.Cinemas {
			if cinema.CinemaId == cinemaId {
				return cinema.CinemaName
			}
		}
	}
	return ""
}
