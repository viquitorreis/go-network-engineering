package main

type Topic string

const (
	BTCTopic Topic = "BTC"
	ETHTopic Topic = "ETH"
)

func (t *Topic) IsValid() bool {
	switch *t {
	case BTCTopic, ETHTopic:
		return true
	default:
		return false
	}
}

func GetTopics() []Topic {
	return []Topic{BTCTopic, ETHTopic}
}
