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

func ParseNetStats(buffer []byte) NetStats {
	stats := NetStats{
		LocalQuality:  math.Float32frombits(binary.BigEndian.Uint32(buffer[0:4])),
		RemoteQuality: math.Float32frombits(binary.BigEndian.Uint32(buffer[4:8])),
		Ping:          int32(binary.BigEndian.Uint32(buffer[8:12])),
		OutByteSec:    math.Float32frombits(binary.BigEndian.Uint32(buffer[12:16])),
		InByteSec:     math.Float32frombits(binary.BigEndian.Uint32(buffer[16:20])),
	}
	return stats
}

func ParsePlayerList(buffer []byte) []PlayerInfo {
	count := int(binary.BigEndian.Uint32(buffer[0:4]))
	result := make([]PlayerInfo, count)

	buffer = buffer[4:]
	for i := range result {
		next := &result[i]
		nameCount := int(binary.BigEndian.Uint32(buffer[0:4]))
		next.Name = string(buffer[4 : nameCount+4])
		buffer = buffer[nameCount+4:]
	}
	return result
}
