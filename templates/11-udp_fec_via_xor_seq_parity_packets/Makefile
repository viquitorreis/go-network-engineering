.PHONY: build-receptor build-client run-receptor run-client

RECEPTOR_BIN=receptor/bin/
SENDER_BIN=sender/bin/

test:
	@go test -v ./receptor/...
	@go test -v ./sender/...

build-receptor:
	@mkdir -p ${RECEPTOR_BIN} 
	@go build -o ${RECEPTOR_BIN}/receptor ./receptor/.

build-sender:
	@mkdir -p ${SENDER_BIN} 
	@go build -o ${SENDER_BIN}/sender ./sender/.

run-receptor: build-receptor
	@./receptor/bin/receptor

run-sender: build-sender
	@./sender/bin/sender