package main

import (
	"encoding/binary"
	"math"
)

type Packet struct {
	Id      int32
	Op      int32
	Payload []byte
}

type PlayerInfo struct {
	Name string
}

type NetStats struct {
	LocalQuality  float32
	RemoteQuality float32
	Ping          int32
	OutByteSec    float32
	InByteSec     float32
}

func ParseString(buffer []byte) (string, []byte) {
	nameLength := binary.BigEndian.Uint32(buffer[0:4])
	nameData := buffer[4 : nameLength+4]
	return string(nameData), buffer[nameLength+4:]
}

func ParseNetStats(buffer []byte) (NetStats, []byte) {
	stats := NetStats{
		LocalQuality:  math.Float32frombits(binary.BigEndian.Uint32(buffer[0:4])),
		RemoteQuality: math.Float32frombits(binary.BigEndian.Uint32(buffer[4:8])),
		Ping:          int32(binary.BigEndian.Uint32(buffer[8:12])),
		OutByteSec:    math.Float32frombits(binary.BigEndian.Uint32(buffer[12:16])),
		InByteSec:     math.Float32frombits(binary.BigEndian.Uint32(buffer[16:20])),
	}
	return stats, buffer[20:]
}

func ParsePlayerList(buffer []byte) ([]PlayerInfo, []byte) {
	count := int(binary.BigEndian.Uint32(buffer[0:4]))
	result := make([]PlayerInfo, count)

	buffer = buffer[4:]
	for i := range result {
		next := &result[i]
		next.Name, buffer = ParseString(buffer)
	}
	return result, buffer
}
