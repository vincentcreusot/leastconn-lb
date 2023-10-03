.PHONY: test test-race simulate-servers stop-servers
test:
	@go test -cover ./...

test-race:
	@go test -race ./...

simulate-servers:
	@cd test && docker-compose up

stop-servers:
	@cd test && docker-compose down



