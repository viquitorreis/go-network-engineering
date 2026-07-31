package protocol

import (
	"math"
	"testing"
)

func TestRTPDatagram_ToDatagram_RoundTrip(t *testing.T) {
	original := Datagram{
		SeqNumber: 42,
		Payload:   1234,
		Timestamp: 1_700_000_000_000,
	}

	encoded := original.ToDatagram()
	parsed := Parse(encoded)

	if parsed == nil {
		t.Fatal("Parse retornou nil pra um datagrama válido")
	}
	if parsed.SeqNumber != original.SeqNumber {
		t.Errorf("SeqNumber = %d, esperado %d", parsed.SeqNumber, original.SeqNumber)
	}
	if parsed.Payload != original.Payload {
		t.Errorf("Payload = %d, esperado %d", parsed.Payload, original.Payload)
	}
	if parsed.Timestamp != original.Timestamp {
		t.Errorf("Timestamp = %d, esperado %d", parsed.Timestamp, original.Timestamp)
	}
}

func TestDatagram_ToDatagram_ZeroValues(t *testing.T) {
	original := Datagram{SeqNumber: 0, Payload: 0, Timestamp: 0}

	parsed := Parse(original.ToDatagram())
	if parsed == nil {
		t.Fatal("Parse não deveria falhar num datagrama com seq/payload/timestamp zerados")
	}
	if parsed.SeqNumber != 0 || parsed.Payload != 0 || parsed.Timestamp != 0 {
		t.Errorf("esperava todos os campos zerados, veio %+v", parsed)
	}
}

func TestDatagram_ToDatagram_MaxValues(t *testing.T) {
	original := Datagram{
		SeqNumber: math.MaxUint32,
		Payload:   math.MaxUint16,
		Timestamp: math.MaxUint64,
	}

	parsed := Parse(original.ToDatagram())
	if parsed == nil {
		t.Fatal("Parse não deveria falhar em valores máximos de cada campo")
	}
	if parsed.SeqNumber != original.SeqNumber {
		t.Errorf("SeqNumber = %d, esperado %d", parsed.SeqNumber, original.SeqNumber)
	}
	if parsed.Payload != original.Payload {
		t.Errorf("Payload = %d, esperado %d", parsed.Payload, original.Payload)
	}
	if parsed.Timestamp != original.Timestamp {
		t.Errorf("Timestamp = %d, esperado %d", parsed.Timestamp, original.Timestamp)
	}
}

func TestParse_RejectsTooShortInput(t *testing.T) {
	// menos bytes do que qualquer datagrama válido poderia ter
	parsed := Parse([]byte{1, 2, 3})
	if parsed != nil {
		t.Errorf("esperava nil pra input muito curto, veio %+v", parsed)
	}
}

func TestParse_RejectsEmptyInput(t *testing.T) {
	parsed := Parse([]byte{})
	if parsed != nil {
		t.Errorf("esperava nil pra input vazio, veio %+v", parsed)
	}
}

func TestDatagram_DifferentSeqNumbers_ProduceDifferentDatagrams(t *testing.T) {
	a := Datagram{SeqNumber: 1, Payload: 100, Timestamp: 1000}
	b := Datagram{SeqNumber: 2, Payload: 100, Timestamp: 1000}

	encodedA := a.ToDatagram()
	encodedB := b.ToDatagram()

	parsedA := Parse(encodedA)
	parsedB := Parse(encodedB)

	if parsedA.SeqNumber == parsedB.SeqNumber {
		t.Error("datagramas com seq numbers diferentes não deveriam parsear pro mesmo seq")
	}
}
