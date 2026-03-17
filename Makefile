.PHONY: build run test clean deps dev all

APP_NAME := goclaw
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

all: build-frontend build

run: build
	./$(BUILD_DIR)/$(APP_NAME) 

dev: build-frontend
	go run ./cmd/server

test:
	go test -v -race ./...

deps:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR) web/dist

lint:
	golangci-lint run ./...

dev-frontend:
	cd web && npm run dev

build-frontend:
	cd web && npm run build
