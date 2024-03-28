include .envrc

# ==================================================================================== #
# HELPERS 
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo 'Are you sure [y/N] \c' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api/ -db-dsn=${FOLLOWUPS_DB_DSN}

## db/psql: connect to the database using psql
.PHONY = db/psql
db/psql:
	psql ${FOLLOWUPS_DB_DSN}

## db/migration/new name=$1: create a new database migration
.PHONY = db/migration/new
db/migration/new:
	@echo "Creating migration files for ${name}..."
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migration/up: apply all up database migrations
.PHONY = db/migration/up
db/migration/up: confirm
	@echo "Running up migrations..."
	migrate -path ./migrations -database ${FOLLOWUPS_DB_DSN} up

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
audit: 
	@echo 'Tidying and verifying module dependencies...' 
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...


# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell  date -u +"%Y-%m-%dT%H:%M:%SZ")
git_description = $(shell  git describe --always --dirty --tag --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'

## build/api: builds binary executable for api application
.PHONY = build/api
build/api:
	@echo "Building binary..."
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api/
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api
