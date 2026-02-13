NAME=anywhere
OUT_DIR=build
TARGET=$(GOOS)-$(GOARCH)$(GOARM)
BIN_DIR=$(OUT_DIR)/$(TARGET)
VERSION?=dev

ifeq ($(GOOS),windows)
  EXT=.exe
  PACK_CMD=zip -9 -r $(NAME)-$(TARGET)-$(VERSION).zip $(TARGET)
else
  EXT=
  PACK_CMD=tar czpvf $(NAME)-$(TARGET)-$(VERSION).tar.gz $(TARGET)
endif

define check_env
	@ if [ "$(GOOS)" = "" ]; then echo " <- Env variable GOOS not set"; exit 1; fi
	@ if [ "$(GOARCH)" = "" ]; then echo " <- Env variable GOARCH not set"; exit 1; fi
endef

.PHONY: build build-dev cross-build release test clean

build: clean test
	@CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(NAME)$(EXT)
	@echo "binary file: $(BIN_DIR)/$(NAME)$(EXT)"

build-dev: clean test
	@go build -o $(BIN_DIR)/$(NAME)$(EXT)
	@echo "binary file: $(BIN_DIR)/$(NAME)$(EXT)"

cross-build: clean test
	@$(call check_env)
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(NAME)$(EXT)
	@echo "binary file: $(BIN_DIR)/$(NAME)$(EXT)"

release:
	@$(call check_env)
	@mkdir -p $(BIN_DIR)
	@cp LICENSE $(BIN_DIR)/
	@cp README.md $(BIN_DIR)/
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(NAME)$(EXT)
	@cd $(OUT_DIR) ; $(PACK_CMD)

test:
	@go test -v -bench=. ./...

clean:
	@go clean
	@rm -rf $(OUT_DIR)
