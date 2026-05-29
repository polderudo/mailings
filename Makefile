-include .env

.PHONY: help test bob gen live clean clean-live restart-live down up seed _templ-live _sync-assets

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@latest
GOOSE_DB ?= mailings
GOOSE_USER ?= mailings
GOOSE_PWD ?= mailings123
GOOSE_PORT ?= 5433
GOOSE_HOST ?= localhost
GOOSE_TO_VERSION ?= 1

BOB_VERSION := 0.44.0
BOB_DSN := "postgres://$(GOOSE_USER):$(GOOSE_PWD)@$(GOOSE_HOST):$(GOOSE_PORT)/$(GOOSE_DB)?sslmode=disable"

BIN_NAME?=app

api:
	go build -o $(BIN_NAME)

test:
	go test ./...

bob-get:
	go get -u github.com/stephenafamo/bob@v$(BOB_VERSION)

bob:
	export PSQL_DSN=$(BOB_DSN) && go run github.com/stephenafamo/bob/gen/bobgen-psql@v$(BOB_VERSION) -c ./db/bobgen.yaml

up:
	$(GOOSE) postgres -table goose_version -dir db/migrations "user=$(GOOSE_USER) password=$(GOOSE_PWD) dbname=$(GOOSE_DB) port=$(GOOSE_PORT) host=$(GOOSE_HOST) sslmode=disable" up

down:
	$(GOOSE) postgres -table goose_version -dir db/migrations "user=$(GOOSE_USER) password=$(GOOSE_PWD) dbname=$(GOOSE_DB) port=$(GOOSE_PORT) host=$(GOOSE_HOST) sslmode=disable" down-to $(GOOSE_TO_VERSION)
