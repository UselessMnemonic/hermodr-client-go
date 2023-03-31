package main

import (
	"encoding/json"
	"github.com/mitchellh/go-ps"
	"strings"
	"sync"
	"sync/atomic"
)

func findValheimProcess() (ps.Process, error) {
	procs, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, v := range procs {
		name := v.Executable()
		if strings.Contains(name, "valheim_server") {
			return v, nil
		}
	}
	return nil, nil
}

type StatusContainer struct {
	json           []byte
	jsonLock       sync.RWMutex
	playerList     []PlayerInfo
	playerListLock sync.RWMutex
	netStats       NetStats
	netStatsLock   sync.RWMutex
	worldName      string
	worldNameLock  sync.RWMutex
	status         string
	statusLock     sync.RWMutex
	startUnix      atomic.Int64
}

func NewStatusContainer() (r StatusContainer) {
	r.startUnix.Store(-1)
	r.worldName = "..."
	r.status = "Unknown"
	r.playerList = make([]PlayerInfo, 0)
	r.updateStatusText(false)
	return
}

func (r *StatusContainer) setPlayerList(list []PlayerInfo) {
	r.playerListLock.Lock()
	r.playerList = list
	r.playerListLock.Unlock()
}

func (r *StatusContainer) GetPlayerList() []PlayerInfo {
	r.playerListLock.RLock()
	defer r.playerListLock.RUnlock()
	return r.playerList
}

func (r *StatusContainer) setWorldName(name string) {
	r.worldNameLock.Lock()
	r.worldName = name
	r.worldNameLock.Unlock()
}

func (r *StatusContainer) GetWorldName() string {
	r.worldNameLock.RLock()
	defer r.worldNameLock.RUnlock()
	return r.worldName
}

func (r *StatusContainer) setNetStats(stats NetStats) {
	r.netStatsLock.Lock()
	r.netStats = stats
	r.netStatsLock.Unlock()
}

func (r *StatusContainer) GetNetStats() NetStats {
	r.netStatsLock.RLock()
	defer r.netStatsLock.RUnlock()
	return r.netStats
}

func (r *StatusContainer) setStatus(status string) {
	r.statusLock.Lock()
	r.status = status
	r.statusLock.Unlock()
}

func (r *StatusContainer) GetStatus() string {
	r.statusLock.RLock()
	defer r.statusLock.RUnlock()
	return r.status
}

func (r *StatusContainer) updateStatusText(valid bool) {
	worldName := r.GetWorldName()
	netStats := r.GetNetStats()
	playerList := r.GetPlayerList()
	status := r.GetStatus()
	jsonStruct := GameStatus{
		Started: 0,
		Status:  status,
	}
	if valid {
		jsonStruct = GameStatus{
			Started: r.startUnix.Load(),
			Status:  status,
			Sections: []GameStatusSection{
				&GameStatusTableSection{
					Name:       "World Stats",
					IsVertical: true,
					Columns: map[string][]any{
						"World Name":        {worldName},
						"Ping":              {netStats.Ping},
						"Input (Byte/sec)":  {netStats.InByteSec},
						"Output (Byte/sec)": {netStats.OutByteSec},
					},
				},
				&GameStatusListSection{
					Name: "Players",
					Items: mapElements(playerList, func(x PlayerInfo) any {
						return x.Name
					}),
				},
			},
		}
	}
	text, _ := json.Marshal(jsonStruct)
	r.jsonLock.Lock()
	r.json = text
	r.jsonLock.Unlock()
}

func (r *StatusContainer) GetStatusText() []byte {
	r.jsonLock.RLock()
	defer r.jsonLock.RUnlock()
	return r.json
}

type GameStatusSection interface {
	GetName() string
	GetType() string
	MarshalJSON() ([]byte, error)
}

type GameStatusTableSection struct {
	Name       string
	IsVertical bool
	Columns    map[string][]any
}

func (s *GameStatusTableSection) GetName() string {
	return s.Name
}

func (s *GameStatusTableSection) GetType() string {
	return "table"
}

func (s *GameStatusTableSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":       s.Name,
		"type":       s.GetType(),
		"isVertical": s.IsVertical,
		"columns":    s.Columns,
	})
}

type GameStatusListSection struct {
	Name  string
	Items []any
}

func (s *GameStatusListSection) GetName() string {
	return s.Name
}

func (s *GameStatusListSection) GetType() string {
	return "list"
}

func (s *GameStatusListSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":  s.Name,
		"type":  s.GetType(),
		"items": s.Items,
	})
}

type GameStatus struct {
	Status   string              `json:"status"`
	Started  int64               `json:"started"`
	Sections []GameStatusSection `json:"sections"`
}
