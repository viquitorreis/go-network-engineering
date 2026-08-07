gen:
	go run github.com/cilium/ebpf/cmd/bpf2go -go-package main -tags linux afxdp xdp_redirect_to_socket.c -- -I. -Iheaders

build:
	@mkdir -p bin
	@go build -o bin/af_xdp .

# before running:

# first created the veth pai:
# sudo ip link add veth0 type veth peer name veth1
# sudo ip link set veth0 up
# sudo ip link set veth1 up

# check if it works
# ip link show veth0
# ip link show veth1

run: build
	@sudo ./bin/af_xdp veth0

# to test:
# sudo ip addr add 10.0.0.1/24 dev veth0
# sudo ip addr add 10.0.0.2/24 dev veth1
#  ping -c 5 10.0.0.1
