IFACE ?= lo
HEADERS := headers

## run with: sudo ./counter lo 9999
## test with: echo "teste" | nc -u -w1 localhost 9999
## 		OR echo teste | nc -u -w1 127.0.0.1 9999

.PHONY: all generate build run clean

all: generate build

generate:
	@go generate ./...

build: generate
	@go build -o counter .

run: build
	@sudo ./counter $(IFACE)

clean:
	@rm -f counter counter_bpfel.go counter_bpfeb.go counter_bpfel.o counter_bpfeb.o xdp_count_by_port.o