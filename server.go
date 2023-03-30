package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"time"
)

var PlayerListRequest = Packet{
	Id:      0,
	Op:      1,
	Payload: make([]byte, 0),
}

var WorldNameRequest = Packet{
	Id:      0,
	Op:      2,
	Payload: make([]byte, 0),
}

var NetStatsRequest = Packet{
	Id:      0,
	Op:      3,
	Payload: make([]byte, 0),
}

var StartTimeRequest = Packet{
	Id:      0,
	Op:      4,
	Payload: make([]byte, 0),
}

var container = NewStatusContainer()

func parseString(buffer []byte) (string, []byte) {
	nameLength := binary.BigEndian.Uint32(buffer[0:3])
	nameData := buffer[4 : nameLength+4]
	return string(nameData), buffer[nameLength+8:]
}

func periodicRequestLoop(requests chan<- Packet, ctx context.Context) {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	requests <- WorldNameRequest
	requests <- StartTimeRequest
	for {
		select {
		case <-timer.C:
			requests <- PlayerListRequest
			requests <- NetStatsRequest
		case <-ctx.Done():
			break
		}
	}
}

func statusUpdaterLoop(responses <-chan Packet, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-responses:
			switch r.Op {
			case 1:
				v := ParsePlayerList(r.Payload)
				container.setPlayerList(v)
				break
			case 2:
				v, _ := parseString(r.Payload)
				container.setWorldName(v)
				break
			case 3:
				v := ParseNetStats(r.Payload)
				container.setNetStats(v)
				break
			case 4:
				v := binary.BigEndian.Uint64(r.Payload)
				container.startUnix.Store(int64(v))
				break
			}
			container.updateStatusText()
		}
	}
}

func updateLoop(ctx context.Context) {
	requests := make(chan Packet, 5)
	responses := make(chan Packet, 5)
	join := make(chan empty, 2)
	for {
		if _, isDone := <-ctx.Done(); isDone {
			fmt.Println("updates canceled, have a nice day")
			return
		}
		proc, err := findValheimProcess()
		if err == nil {
			fmt.Printf("error while enumerating processes: %e", err)
		}
		if proc == nil {
			fmt.Println("game server not found, try again in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
		}
		container.setStatus("Running")
		client, err := DialHermodr(":2458")
		if err != nil {
			fmt.Printf("error while dialing game server: %e\n", err)
			continue
		}
		innerCtx, innerCancel := context.WithCancel(ctx)
		go periodicRequestLoop(requests, innerCtx)
		go statusUpdaterLoop(responses, innerCtx)
		go func() {
			for {
				pkt, err := client.Receive()
				if err != nil {
					join <- empty{}
					return
				}
				responses <- pkt
			}
		}()
		go func() {
			for {
				pkt := <-requests
				err := client.Send(pkt)
				if err != nil {
					join <- empty{}
					return
				}
			}
		}()
		<-join
		<-join
		log.Println("client disconnected")
		innerCancel()
		_ = client.Close()
	}
}

func handleStatus(writer http.ResponseWriter, _ *http.Request) {
	statusText := container.GetStatusText()
	_, _ = writer.Write(statusText)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := http.NewServeMux()
	handleRedirect := http.RedirectHandler("https://status.uselessmnemonic.com", http.StatusSeeOther)
	router.Handle("/", handleRedirect)
	router.HandleFunc("/status", handleStatus)

	server := &http.Server{
		Addr:    ":80",
		Handler: router,
	}

	go updateLoop(ctx)
	_ = server.ListenAndServe()
}
