package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type HermodrClient struct {
	conn net.Conn
}

func DialHermodr(address string) (*HermodrClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &HermodrClient{
		conn: conn,
	}, nil
}

func (provider *HermodrClient) Close() error {
	return provider.conn.Close()
}

func (provider *HermodrClient) Receive() (Packet, error) {
	headerBuffer := [12]byte{}
	actual, err := io.ReadFull(provider.conn, headerBuffer[:])
	if err != nil {
		return Packet{}, err
	}
	if actual != len(headerBuffer) {
		return Packet{}, fmt.Errorf("expected %d bytes but got %d", len(headerBuffer), actual)
	}

	response := Packet{}
	response.Id = int32(binary.BigEndian.Uint32(headerBuffer[0:4]))
	response.Op = int32(binary.BigEndian.Uint32(headerBuffer[4:8]))
	payloadLen := int32(binary.BigEndian.Uint32(headerBuffer[8:12]))
	response.Payload = make([]byte, payloadLen)
	actual, err = io.ReadFull(provider.conn, response.Payload)
	if err != nil {
		return Packet{}, err
	}
	if actual != int(payloadLen) {
		return Packet{}, fmt.Errorf("expected %d bytes but got %d", payloadLen, actual)
	}
	return response, nil
}

func (provider *HermodrClient) Send(request Packet) error {
	headerBuffer := [12]byte{}
	binary.BigEndian.PutUint32(headerBuffer[0:4], uint32(request.Id))
	binary.BigEndian.PutUint32(headerBuffer[4:8], uint32(request.Op))
	binary.BigEndian.PutUint32(headerBuffer[8:12], uint32(len(request.Payload)))
	actual, err := provider.conn.Write(headerBuffer[:])
	if err != nil {
		return err
	}
	if actual != len(headerBuffer) {
		return fmt.Errorf("had %d bytes but sent %d", len(headerBuffer), actual)
	}

	actual, err = provider.conn.Write(request.Payload)
	if err != nil {
		return err
	}
	if expected := len(request.Payload); expected != actual {
		return fmt.Errorf("expected %d bytes but got %d", expected, actual)
	}
	return nil
}
