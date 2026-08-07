gen:
	go run github.com/cilium/ebpf/cmd/bpf2go -go-package main -tags linux afxdp xdp_redirect_to_socket.c -- -I. -Iheaders

build:
	@mkdir -p bin
	@go build -o bin/af_xdp .

run: build
	@sudo ./bin/af_xdp veth0