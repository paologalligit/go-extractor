package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/paologalligit/go-extractor/entities"
)

func WriteResultsToFile(results []entities.ShowingResult, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	fmt.Printf("\nDone! Results written to %s\n", filename)
	return nil
}
