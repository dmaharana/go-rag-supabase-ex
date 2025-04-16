# Makefile to build for different OS

APP_NAME := gorag
SRC := ./cmd/main.go
BUILD_DIR := build
LDFLAGS := -s -w

UNAME := $(shell uname)

ifeq ($(UNAME), Linux)
	BUILD_TARGET = linux
endif
ifeq ($(UNAME), Darwin)
	BUILD_TARGET = darwin
endif
ifeq ($(UNAME), Windows_NT)
	BUILD_TARGET = windows
endif

.PHONY: all clean

all: $(BUILD_TARGET)

linux l:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux $(SRC)

windows w:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows.exe $(SRC)

darwin m:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin $(SRC)

all: linux windows darwin

clean:
	rm -rf $(BUILD_DIR)

