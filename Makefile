APP_NAME=docker-volume-backup

all: build

build:
	CGO_ENABLED=0 go build -o $(APP_NAME) ./cmd

install:
	sudo mv $(APP_NAME) /usr/local/bin/$(APP_NAME)

clean:
	rm -f $(APP_NAME)
	rm -rf dist

test:
	go vet ./...
	go test ./...
	go test -tags=integration ./...

# cross-platform build - static binaries
release:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/$(APP_NAME)-linux-amd64 ./cmd
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/$(APP_NAME)-linux-arm64 ./cmd
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o dist/$(APP_NAME)-darwin-amd64 ./cmd
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o dist/$(APP_NAME)-darwin-arm64 ./cmd
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/$(APP_NAME)-windows-amd64.exe ./cmd
