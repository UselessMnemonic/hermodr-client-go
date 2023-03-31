package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
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
	var remainingPayload []byte
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-responses:
			switch r.Op {
			case 1:
				var v []PlayerInfo
				v, remainingPayload = ParsePlayerList(r.Payload)
				container.setPlayerList(v)
				break
			case 2:
				var v string
				v, remainingPayload = ParseString(r.Payload)
				container.setWorldName(v)
				break
			case 3:
				var v NetStats
				v, remainingPayload = ParseNetStats(r.Payload)
				container.setNetStats(v)
				break
			case 4:
				var v uint64
				v = binary.BigEndian.Uint64(r.Payload)
				remainingPayload = r.Payload[8:]
				container.startUnix.Store(int64(v))
				break
			}
			container.updateStatusText()
		}
		if rem := len(remainingPayload); rem > 0 {
			fmt.Printf("%d bytes remaining in payload!!!", rem)
		}
	}
}

func updateLoop(ctx context.Context) {
	requests := make(chan Packet, 5)
	responses := make(chan Packet, 5)
	join := make(chan empty, 2)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("updates canceled, have a nice day")
			return
		default:
			break
		}
		fmt.Println("searching for game process")
		proc, err := findValheimProcess()
		if err != nil {
			container.setStatus("Unknown")
			container.updateStatusText()
			fmt.Printf("error while enumerating processes: %e\n", err)
		}
		if proc == nil {
			container.setStatus("Stopped")
			container.updateStatusText()
			fmt.Println("game server not found")
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("dialing gamer server...")
		client, err := DialHermodr(":2458")
		if err != nil {
			container.setStatus("Starting")
			container.updateStatusText()
			fmt.Printf("error while dialing game server: %e\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		container.setStatus("Running")
		container.updateStatusText()
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

	policy := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	server := &http.Server{
		Addr:    ":443",
		Handler: policy.Handler(router),
	}

	fullCert, ok := os.LookupEnv("FULL_CERT_PATH")
	if !ok {
		fmt.Printf("full cert path not specified")
	}
	privKey, ok := os.LookupEnv("PRIV_KEY_PATH")
	if !ok {
		fmt.Printf("private key path not specified")
	}

	fmt.Println("starting server...")
	go updateLoop(ctx)
	err := server.ListenAndServeTLS(fullCert, privKey)
	fmt.Printf("main done: %e\n", err)
}
