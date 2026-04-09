GOCACHE ?= /tmp/go-build
BINARY ?= transport
DEPLOY_DIR ?= /opt/proletarka-transport
DEPLOY_BIN ?= $(DEPLOY_DIR)/$(BINARY)
SYSTEMD_SERVICE ?= proletarka-transport

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
	GOCACHE=$(GOCACHE) go build -o $(BINARY) ./cmd/transport
	install -d $(DEPLOY_DIR)
	install -m 755 $(BINARY) $(DEPLOY_BIN)
	systemctl restart $(SYSTEMD_SERVICE)
	systemctl status $(SYSTEMD_SERVICE) --no-pager
