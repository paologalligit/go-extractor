# go_extractor

## Overview

go_extractor is a modular Go project for extracting, scheduling, and logging cinema showings and seat occupancy from The Space Cinema APIs. It is designed for reliability, extensibility, and clear separation of concerns.

**Key Features:**
- Fetches all showings for a given day/cinema/film and writes them to a file
- Schedules timers to query and log seat counts for each session after it starts
- Modular codebase with clear package boundaries
- Append-only logging for auditability

---

## Architecture & Packages

- **main.go**: CLI entry point, command dispatch, and high-level workflow
- **fetchshowings/**: Fetches showings and writes them to a file (see `fetchshowings.go`)
- **settimers/**: Reads a sessions file and schedules seat counting timers (see `settimers.go`)
- **entities/**: All core data structures (cinema, film, session, showing, etc.)
- **header/**: Cookie and header management for authenticated requests
- **constant/**: API endpoint constants
- **utils/**: Utility functions (e.g., file helpers)

---

## File Formats

- **Showings file** (`showings_YYYYMMDD_HHMMSS.json` or `todaySession-YYYY-MM-DD.json`):
  - Array of `ScheduledSession` objects (see `entities/entities.go`)
  - Each entry contains all info needed to schedule and query a session

- **Seat counts log** (`seat_counts.log`):
  - Append-only, one JSON object per line
  - Each entry: cinema, film, session, seat count, start hour, timestamp

---

## Commands & Usage

### 1. Fetch Showings
Fetches all showings for a given day and writes them to a file.

**Usage:**
```sh
go run main.go fetch-showings --workers=10 --delay=100
```
- Options:
  - `--workers`: Number of concurrent workers (default: 10)
  - `--delay`: Delay between requests in milliseconds (default: 100)
- Output: `showings_YYYYMMDD_HHMMSS.json` or `todaySession-YYYY-MM-DD.json`

### 2. Seat Timers
Reads a sessions file and schedules timers for each session. After each session starts, it queries the seat count and logs the result.

**Usage:**
```sh
go run main.go seat-timers
```
- If `todaySession-YYYY-MM-DD.json` does not exist, it will be created automatically for today.
- Output: `seat_counts.log`

---

## Example Workflow

1. **Fetch showings for the day:**
   ```sh
   go run main.go fetch-showings --workers=20 --delay=200
   # Creates a file like showings_20250915_130905.json or todaySession-2025-09-15.json
   ```

2. **Start seat counting timers:**
   ```sh
   go run main.go seat-timers
   # Reads the sessions file and logs seat counts as sessions start
   ```

---

## Extensibility & Notes
- The codebase is modular: add new fetchers, loggers, or timer strategies easily.
- All API endpoints and constants are centralized in the `constant/` package.
- Cookie and header management is handled in the `header/` package using Playwright for robust authentication.
- All logs are append-only for auditability and post-processing.
- Error handling is robust and all errors are logged with context.
- The project is ready for further automation, scheduling, or integration with other systems.

---

## Contributing
- Pull requests and issues are welcome!
- Please keep code modular and document new features or changes in the README.

---

## Maintainers
- [Your Name or Team]

---

## License
[Specify your license here]
