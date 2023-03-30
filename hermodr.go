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
	n, err := io.ReadFull(provider.conn, headerBuffer[:])
	if err != nil {
		return Packet{}, err
	}
	if n != len(headerBuffer) {
		return Packet{}, fmt.Errorf("expected %d bytes but got %d", len(headerBuffer), n)
	}

	response := Packet{}
	response.Id = int32(binary.BigEndian.Uint32(headerBuffer[0:4]))
	response.Op = int32(binary.BigEndian.Uint32(headerBuffer[4:8]))
	payloadLen := int32(binary.BigEndian.Uint32(headerBuffer[8:12]))
	response.Payload = make([]byte, payloadLen)
	n, err = io.ReadFull(provider.conn, response.Payload)
	if err != nil {
		return Packet{}, err
	}
	if n != len(response.Payload) {
		return Packet{}, fmt.Errorf("expected %d bytes but got %d", len(headerBuffer), n)
	}
	return response, nil
}

func (provider *HermodrClient) Send(request Packet) error {
	headerBuffer := [12]byte{}
	binary.BigEndian.PutUint32(headerBuffer[0:4], uint32(request.Id))
	binary.BigEndian.PutUint32(headerBuffer[4:8], uint32(request.Op))
	binary.BigEndian.PutUint32(headerBuffer[8:12], uint32(len(request.Payload)))
	n, err := provider.conn.Write(headerBuffer[:])
	if err != nil {
		return err
	}
	if n != len(headerBuffer) {
		return fmt.Errorf("had %d bytes but sent %d", len(headerBuffer), n)
	}

	n, err = provider.conn.Write(request.Payload)
	if err != nil {
		return err
	}
	if n != len(request.Payload) {
		return fmt.Errorf("expected %d bytes but got %d", len(headerBuffer), n)
	}
	return nil
}
