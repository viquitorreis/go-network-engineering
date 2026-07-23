package protocol

// datagram represents a packet of data,
// each packet will represent a frame to be played on a specific time frame
// the seq number solves de sequence detection problem
type RTPDatagram struct {
}

func Parse(b []byte) *RTPDatagram {

}

func (r RTPDatagram) ToDatagram() []byte {
}
