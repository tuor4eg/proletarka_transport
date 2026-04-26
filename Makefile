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

sync-env:
	@test -f "$(ENV_EXAMPLE)" || (echo "$(ENV_EXAMPLE) not found" && exit 1)
	@touch "$(ENV_FILE)"
	@awk -F= '\
		FNR == NR {\
			if ($$0 ~ /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=/) {\
				key = $$1;\
				gsub(/^[[:space:]]+|[[:space:]]+$$/, "", key);\
				existing[key] = 1;\
			}\
			next;\
		}\
		$$0 ~ /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=/ {\
			key = $$1;\
			gsub(/^[[:space:]]+|[[:space:]]+$$/, "", key);\
			if (!(key in existing)) {\
				print key "=";\
				added += 1;\
			}\
		}\
		END {\
			if (added > 0) {\
				printf("Added %d missing env variable(s) to $(ENV_FILE)\n", added) > "/dev/stderr";\
			} else {\
				printf("$(ENV_FILE) already has all variables from $(ENV_EXAMPLE)\n") > "/dev/stderr";\
			}\
		}\
	' "$(ENV_FILE)" "$(ENV_EXAMPLE)" >> "$(ENV_FILE)"