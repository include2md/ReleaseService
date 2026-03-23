COMPOSE ?= docker compose
API ?= http://localhost:8080
TOKEN ?= local-dev-token
APP ?= my-app
VERSION ?= 1.2.3
ENV ?= prod
ARTIFACT ?= sample.tar.gz

.PHONY: up down logs ps test test-upload

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

test:
	go test ./...

test-upload:
	@test -f "$(ARTIFACT)" || (echo "artifact file not found: $(ARTIFACT)" && exit 1)
	curl -X POST '$(API)/api/v1/releases' \
		-H 'Authorization: Bearer $(TOKEN)' \
		-F 'app_name=$(APP)' \
		-F 'version=$(VERSION)' \
		-F 'environment=$(ENV)' \
		-F 'artifact=@$(ARTIFACT)'
