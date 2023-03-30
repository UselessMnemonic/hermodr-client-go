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
		LocalQuality:  math.Float32frombits(binary.BigEndian.Uint32(buffer[0:3])),
		RemoteQuality: math.Float32frombits(binary.BigEndian.Uint32(buffer[4:7])),
		Ping:          int32(binary.BigEndian.Uint32(buffer[8:11])),
		OutByteSec:    math.Float32frombits(binary.BigEndian.Uint32(buffer[12:15])),
		InByteSec:     math.Float32frombits(binary.BigEndian.Uint32(buffer[16:19])),
	}
	return stats
}

func ParsePlayerList(buffer []byte) []PlayerInfo {
	count := int(binary.BigEndian.Uint32(buffer[0:3]))
	result := make([]PlayerInfo, count)
	if count == 0 {
		return result
	}

	buffer = buffer[4:]
	for _, next := range result {
		nameCount := int(binary.BigEndian.Uint32(buffer[0:3]))
		next.Name = string(buffer[4 : nameCount+3])
		buffer = buffer[nameCount+3:]
	}
	return result
}
