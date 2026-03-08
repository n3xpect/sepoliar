MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

-include $(MAKEFILE_DIR)/../.env
export

IMAGE_NAME := sepoliar
IMAGE_TAG  := 1.0.0


build:
	cd $(MAKEFILE_DIR)/.. && CGO_ENABLED=0 go build -buildvcs=false -ldflags="-w -s" -o sepoliar .

docker-build:
	cd $(MAKEFILE_DIR)/.. && docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f deploy/Dockerfile .

compose-up:
	docker compose -f $(MAKEFILE_DIR)/docker-compose.yaml up -d && docker compose -f $(MAKEFILE_DIR)/docker-compose.yaml logs -f



compose-down:
	docker compose -f $(MAKEFILE_DIR)/docker-compose.yaml down



railway-env-set:
	railway variables set \
		LOG_LEVEL=$(LOG_LEVEL) \
		TZ=$(TZ) \
		TELEGRAM_BOT_TOKEN=$(TELEGRAM_BOT_TOKEN) \
		TELEGRAM_CHAT_ID=$(TELEGRAM_CHAT_ID) \
		WALLET_ADDRESS_ETH=$(WALLET_ADDRESS_ETH)

railway-env-list:
	railway variables

railway-up:
	cd $(MAKEFILE_DIR)/.. && railway up --dockerfile deploy/Dockerfile
