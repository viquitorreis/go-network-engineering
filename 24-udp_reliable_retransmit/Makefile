.PHONY: build-server build-client run-server run-client

SERVER_BIN=server/bin/
CLIENT_BIN=client/bin/

test:
	@go test -v ./server/...
	@go test -v ./client/...

test-bench:
	@go test -bench=BenchmarkReliability -benchtime=200x -v ./... 2>&1 | grep -E "loss_|PASS|FAIL"
	
build-server:
	@mkdir -p ${SERVER_BIN} 
	@go build -o ${SERVER_BIN}/server ./server/.

build-client:
	@mkdir -p ${CLIENT_BIN} 
	@go build -o ${CLIENT_BIN}/client ./client/.

run-server: build-server
	@./server/bin/server

run-client: build-client
	@./client/bin/client