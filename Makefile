WORKERS ?= 10
DELAY ?= 100

all:
	go run main.go --workers=$(WORKERS) --delay=$(DELAY) all

today:
	docker compose up -d
	go run main.go --workers=$(WORKERS) --delay=$(DELAY) today
	docker compose down

initdb:
	docker compose up -d
	go run main.go initdb
	docker compose down