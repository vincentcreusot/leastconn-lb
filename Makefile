.PHONY: build-balancer test-balancer

build-balancer:
	@go build -o bin/balancer balancer/main.go

test-balancer:
	@go test -cover ./balancer/...

test-balancer-race:
	@go test -race ./balancer/forwarder/...