WORKERS ?= 10
DELAY ?= 100

all:
	go run main.go --workers=$(WORKERS) --delay=$(DELAY) all
	rm -rf showings_*.json

today:
	docker compose up -d
	go run main.go --workers=$(WORKERS) --delay=$(DELAY) today
	docker compose down
	rm -rf todaySession-*.json
	osascript -e 'tell application "System Events" to shut down'

initdb:
	docker compose up -d
	go run main.go initdb
	docker compose down