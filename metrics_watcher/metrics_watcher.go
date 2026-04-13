package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

/* determina automaticamente il path corretto (/proc o /host/proc, in modo tale che
il programma funzione sia dentro kubernetes che fuori ( dentro kubernetes viene montato
l'hostpath e quindi il path completo sarà /host/proc e non solo /proc) */

var procPath = getProcPath()

func getProcPath() string {
	if _, err := os.Stat("/host/proc"); err == nil { //os.Stat ci ridà le infrmazione di una cartella ( con _, err igoriamo il risultato ma non l'eventuale errore)
		return "/host/proc" // questo if segue la struttura go if <inizializzazione>; <condizione> { con doppia inizializzazione
	}
	return "/proc"
}

type Metrics struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"` // già convertito in percentuale
	MemTotalKB      uint64  `json:"mem_total_kb"`
	MemUsedKB       uint64  `json:"mem_used_kb"`
	MemAvailableKB  uint64  `json:"mem_available_kb"`
	NodeIP          string  `json:"node_ip"`
}

// Legge CPU da /proc/stat
// restituisce float64 e una variabile error
func getCPUUsage() (float64, error) {

	// idle=tempo idle, total = tempo totale
	readCPU := func() (idle, total uint64, err error) { //assegno alla variabile readCPU una funzione

		file, err := os.Open(procPath + "/stat") // apre il file stat
		if err != nil {
			return // quindi ritornerà 0 0 perchè in go le variabili sono automaticamente inizializzate
		}

		defer file.Close() // defer significa esegui la funzione file.Close alla fine della funzione attuale (questa anonima)
		// in generale, garantiamo la chiusura del file alla fine della funzione anonima.

		scanner := bufio.NewScanner(file) // apriamo uno scanner, e se non si apre ritorniamo 0, 0  e l'errore dello scanner
		if !scanner.Scan() {
			return 0, 0, scanner.Err()
		}

		fields := strings.Fields(scanner.Text()) // restituisce la riga appena letta (fields è un vettore)
		if len(fields) < 5 {                     // che per la struttura del file deve avere almeno 5 campi
			return 0, 0, err
		}

		var values []uint64            // array dinamico di numeri interi senza segni
		for _, v := range fields[1:] { // cicli su tutti gli elementi tranne il primo (che nel file è la parola cpu)
			val, err := strconv.ParseUint(v, 10, 64) //	 conversione da strign a numero (10 la base, 64 la dimensione (uint64))
			if err != nil {
				return 0, 0, err
			}
			values = append(values, val) // aggiungi il numero
		}

		idle = values[3] // il quarto elemeno dell'array è il tempo idle
		for _, v := range values {
			total += v // somma tutti i valori
		}
		return // ritorna i risultati (in modo implicito, per via della firma della funzione anonima) e chiudi il file (che avevi chiuso con defer )
	}

	// total e idle sono dei contatori che contano da quando il nodo è acceso, una sola lettura decontestualizzata è inutile, per cui
	// facciamo due letture e facciamo la differenza per ottenere i tempi di idle e total in un piccolo lasso di tempo e poi calcoliamo in percentuale il
	// rapporto tra il tempo idle e il tempo totale
	idle1, total1, err := readCPU()
	if err != nil {
		return 0, err
	}

	time.Sleep(500 * time.Millisecond)

	idle2, total2, err := readCPU()
	if err != nil {
		return 0, err
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0, nil
	}

	cpuUsage := 1.0 - (idleDelta / totalDelta) // fai il complementare, ergo se esce 0.8 significa che la cpu è usata all'80%
	return cpuUsage * 100, nil                 // ritorna già convertito in percentuale
}

// Legge memoria da /proc/meminfo
// ritorna: memoria totale, memoria usata, memoria disponibile
func getMemory() (total, used, available uint64, err error) {

	file, err := os.Open(procPath + "/meminfo") // stesso pattern di prima
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file) // leggerà riga per riga

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "MemTotal") {
			fields := strings.Fields(line)
			total, _ = strconv.ParseUint(fields[1], 10, 64)
		}

		if strings.HasPrefix(line, "MemAvailable") {
			fields := strings.Fields(line)
			available, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}

	used = total - available
	return
}

// Handler HTTP
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	cpu, err := getCPUUsage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	total, used, available, err := getMemory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeIP := os.Getenv("NODE_IP")
	metrics := Metrics{
		CPUUsagePercent: cpu,
		MemTotalKB:      total,
		MemUsedKB:       used,
		MemAvailableKB:  available,
		NodeIP:          nodeIP,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func main() {
	//registra una route, associa a /metric la funzione metricHandler
	http.HandleFunc("/metrics", metricsHandler)

	log.Println("Server running on :8080")

	// vera parte sospensiva che mette il server in ascolto
	log.Fatal(http.ListenAndServe(":8080", nil))
}
