package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/fetchshowings"
	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/settimers"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [fetch-showings|seat-timers] [options]")
		os.Exit(1)
	}

	cookiesManager, err := header.New()
	if err != nil {
		fmt.Printf("error getting cookies: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch-showings":
		// Parse command line flags
		maxGoroutines := flag.Int("workers", 10, "Number of concurrent workers")
		requestDelay := flag.Int("delay", 100, "Delay between requests in milliseconds")
		flag.Parse()

		fmt.Printf("Configuration: Using %d workers with %dms delay between requests\n", *maxGoroutines, *requestDelay)

		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%s.json", "showings", timestamp)
		opt := &fetchshowings.FetchShowingsOptions{
			MaxGoroutines:  *maxGoroutines,
			RequestDelay:   *requestDelay,
			ShowingUrl:     constant.SHOWINGS_URL,
			OutputFileName: filename,
			CookiesManager: cookiesManager,
		}
		if err := fetchshowings.RunFetchShowings(opt); err != nil {
			fmt.Printf("error running fetch showings: %v\n", err)
			os.Exit(1)
		}
	case "seat-timers":
		opt := &settimers.SettimersOptions{
			CookiesManager: cookiesManager,
		}
		if err := settimers.RunSeatTimers(opt); err != nil {
			fmt.Printf("error running seat timers: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		fmt.Println("Usage: go run main.go [fetch-showings|seat-timers] [options]")
		os.Exit(1)
	}
}
