package main

import "errors"

var (
	ErrInvalidTopic    = errors.New("err invalid topic")
	ErrInvalidPayload  = errors.New("err invalid payload")
	ErrReadingFromConn = errors.New(" reading from conn")
)
