package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// stessa struct degli agent
type Metrics struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemTotalKB      uint64  `json:"mem_total_kb"`
	MemUsedKB       uint64  `json:"mem_used_kb"`
	MemAvailableKB  uint64  `json:"mem_available_kb"`
	NodeIP          string  `json:"node_ip"`
}

// handler del collector
func collectHandler(w http.ResponseWriter, r *http.Request) {

	// 1. Scopri i pod tramite DNS (headless service)
	ips, err := net.LookupHost("node-metrics.default.svc.cluster.local")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []Metrics

	// se la richiesta dura più di 2 secondi, fallisci
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	// 2. Chiama ogni pod (sequenziale)
	for _, ip := range ips {
		url := fmt.Sprintf("http://%s:8080/metrics", ip)

		resp, err := client.Get(url)
		if err != nil {
			log.Println("errore chiamando", ip, err)
			continue
		}

		var m Metrics
		err = json.NewDecoder(resp.Body).Decode(&m)
		resp.Body.Close()

		if err != nil {
			log.Println("errore parsing", ip, err)
			continue
		}

		results = append(results, m)
	}

	// 3. ritorna tutte le metriche (lista)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	http.HandleFunc("/collect", collectHandler)

	log.Println("Collector running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
