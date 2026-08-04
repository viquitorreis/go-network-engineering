IFACE ?= lo
HEADERS := headers

.PHONY: all generate build run clean

all: generate build

generate:
	go generate ./...

build: generate
	go build -o counter .

run: build
	sudo ./counter $(IFACE)

clean:
	rm -f counter counter_bpfel.go counter_bpfeb.go counter_bpfel.o counter_bpfeb.o xdp_count_by_port.o