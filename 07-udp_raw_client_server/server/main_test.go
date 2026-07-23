package main

import (
	"net"
	"testing"
)

func TestServer_AddClient_DedupesSameAddress(t *testing.T) {
	s := NewServer()

	addr1 := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5000}
	addr2 := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5000} // same IP:port, different structs

	s.addClient(addr1)
	s.addClient(addr2)

	if len(s.Clients) != 1 {
		t.Fatalf("esperava 1 client único, tem %d — dedup por addr.String() falhou", len(s.Clients))
	}
}

func TestServer_AddClient_TracksMultipleDistinctAddresses(t *testing.T) {
	s := NewServer()

	addrs := []net.UDPAddr{
		{IP: net.ParseIP("127.0.0.1"), Port: 5000},
		{IP: net.ParseIP("127.0.0.1"), Port: 5001},
		{IP: net.ParseIP("192.168.0.10"), Port: 5000},
	}

	for _, a := range addrs {
		s.addClient(a)
	}

	if len(s.Clients) != 3 {
		t.Fatalf("esperava 3 clients distintos, tem %d", len(s.Clients))
	}

	for _, a := range addrs {
		if _, ok := s.Clients[a.String()]; !ok {
			t.Errorf("client %s não encontrado no map", a.String())
		}
	}
}

func TestServer_AddClient_SamePortDifferentIP_TreatedAsDistinct(t *testing.T) {
	s := NewServer()

	s.addClient(net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 9000})
	s.addClient(net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 9000})

	if len(s.Clients) != 2 {
		t.Fatalf("IPs diferentes na mesma porta deveriam contar como clients distintos, mas deu %d", len(s.Clients))
	}
}
