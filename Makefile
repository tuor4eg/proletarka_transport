GOCACHE ?= /tmp/go-build
BINARY ?= transport
DEPLOY_DIR ?= /opt/proletarka_transport
DEPLOY_BIN ?= $(DEPLOY_DIR)/$(BINARY)
DEPLOY_TMP_BIN ?= $(DEPLOY_BIN).new
SYSTEMD_SERVICE ?= proletarka-transport
DEPLOY_BRANCH ?= main

.PHONY: build run test tidy fmt clean deploy

build:
	GOCACHE=$(GOCACHE) go build -o $(BINARY) ./cmd/transport

run:
	GOCACHE=$(GOCACHE) go run ./cmd/transport

test:
	GOCACHE=$(GOCACHE) go test ./...

tidy:
	GOCACHE=$(GOCACHE) go mod tidy

fmt:
	gofmt -w $(shell find cmd internal -name '*.go' -type f)

clean:
	rm -f $(BINARY)

deploy:
	git pull --ff-only origin $(DEPLOY_BRANCH)
	install -d $(DEPLOY_DIR)
	GOCACHE=$(GOCACHE) go build -o $(DEPLOY_TMP_BIN) ./cmd/transport
	mv $(DEPLOY_TMP_BIN) $(DEPLOY_BIN)
	chmod 755 $(DEPLOY_BIN)
	systemctl restart $(SYSTEMD_SERVICE)
	systemctl status $(SYSTEMD_SERVICE) --no-pager
