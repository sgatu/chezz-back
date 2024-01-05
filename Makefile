build:
	@rm -rf ./dist
	@mkdir ./dist
	cp .env.dev ./dist/.env
	go build -o ./dist/chezz
	@chmod +x ./dist/chezz
run:
	@cd dist; ./chezz
run-release:
	@cd dist; GIN_MODE=release ./chezz
run-console:
	@cd dist; ./chezz console
test:
	go clean -testcache
	go test ./...
test-debug:
	go clean -testcache
	go test -v ./...
