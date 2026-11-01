COMPOSE ?= docker compose

.PHONY: help up down up-replica down-replica up-sharded down-sharded up-loadtest down-loadtest reseed smoke load-1000 stress-10000 spike tidy vet build-frontend logs clean

help:
	@echo "make up            standalone MongoDB stack (unoptimized demo)"
	@echo "make up-replica    3-node replica set stack"
	@echo "make up-sharded    sharded cluster stack"
	@echo "make up-loadtest   standalone stack plus Prometheus and Grafana"
	@echo "make smoke | load-1000 | stress-10000 | spike   run a k6 scenario"
	@echo "make tidy          generate backend/go.sum using a Go container"
	@echo "make vet           go vet inside a Go container"
	@echo "make clean         stop every stack and delete its volumes"

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

up-replica:
	$(COMPOSE) -f docker-compose.replica.yml up --build -d

down-replica:
	$(COMPOSE) -f docker-compose.replica.yml down

up-sharded:
	$(COMPOSE) -f docker-compose.sharded.yml up --build -d

down-sharded:
	$(COMPOSE) -f docker-compose.sharded.yml down

up-loadtest:
	$(COMPOSE) -f docker-compose.loadtest.yml up --build -d --scale k6=0

down-loadtest:
	$(COMPOSE) -f docker-compose.loadtest.yml down

reseed:
	SEED_RESET=true $(COMPOSE) run --rm seed

smoke:
	K6_SCRIPT=smoke.js $(COMPOSE) -f docker-compose.loadtest.yml run --rm k6

load-1000:
	K6_SCRIPT=load-1000.js $(COMPOSE) -f docker-compose.loadtest.yml run --rm k6

stress-10000:
	K6_SCRIPT=stress-10000.js $(COMPOSE) -f docker-compose.loadtest.yml run --rm k6

spike:
	K6_SCRIPT=spike.js $(COMPOSE) -f docker-compose.loadtest.yml run --rm k6

tidy:
	docker run --rm -v "$(CURDIR)/backend:/src" -w /src golang:1.24-alpine3.21 go mod tidy

vet:
	docker run --rm -v "$(CURDIR)/backend:/src" -w /src golang:1.24-alpine3.21 sh -c "go mod tidy && go vet ./..."

build-frontend:
	cd frontend && npm ci && npm run build

logs:
	$(COMPOSE) logs -f api

clean:
	-$(COMPOSE) down -v
	-$(COMPOSE) -f docker-compose.replica.yml down -v
	-$(COMPOSE) -f docker-compose.sharded.yml down -v
	-$(COMPOSE) -f docker-compose.loadtest.yml down -v
