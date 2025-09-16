WORKERS ?= 10
DELAY ?= 100

all:
	go run main.go all --workers=$(WORKERS) --delay=$(DELAY)

today:
	docker compose up -d
	go run main.go today --workers=$(WORKERS) --delay=$(DELAY)
	docker compose down

initdb:
	docker compose up -d
	go run main.go initdb
	docker compose down