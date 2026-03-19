-include .env
export


IMAGE_NAME := sepoliar
IMAGE_TAG  := 1.0.0

.PHONY: build docker-build docker-up docker-up-mac docker-down docker-logs sync
.DEFAULT_GOAL := build


build:
	CGO_ENABLED=0 go build -buildvcs=false -ldflags="-w -s" -o sepoliar .


docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f deploy/Dockerfile .

docker-up:
	chown -R 1000:1000 data
	chmod 755 data/account
	chmod 644 data/account/*.enc 2>/dev/null || true
	docker compose -f deploy/docker-compose.yml up -d
	docker compose -f deploy/docker-compose.yml logs -f

docker-up-mac:
	docker compose -f deploy/docker-compose.mac.yml up -d
	docker compose -f deploy/docker-compose.mac.yml logs -f

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker-compose.yml logs -f

sync:
	rsync -avz --progress \
		--exclude='.git' \
		--exclude='.gitignore' \
		--exclude='.idea' \
		--exclude='.claude' \
		--exclude='railway.toml' \
		--exclude='sepoliar' \
		../sepoliar/ $(SEPOLIAR_SERVER)/sepoliar
