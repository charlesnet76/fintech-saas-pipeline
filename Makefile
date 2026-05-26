.PHONY: help up down test lint build

help:
	@echo "Available commands:"
	@echo "  make up       - Start local dev environment"
	@echo "  make down     - Stop local dev environment"
	@echo "  make test     - Run all tests"
	@echo "  make lint     - Run linters"
	@echo "  make build    - Build all services"

up:
	docker compose -f infrastructure/docker/docker-compose.yml up -d

down:
	docker compose -f infrastructure/docker/docker-compose.yml down

test:
	@for svc in services/*/; do echo "Testing $$svc"; done

lint:
	@for svc in services/*/; do echo "Linting $$svc"; done

build:
	@for svc in services/*/; do echo "Building $$svc"; done
