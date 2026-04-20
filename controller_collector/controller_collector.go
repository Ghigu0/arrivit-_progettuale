package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// indirizzo del proxy esposto dal service
const proxyURL = "http://proxy-service:8080"

// struttura dati delle metriche, utilizzata dai watcher
type Metrics struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemTotalKB      uint64  `json:"mem_total_kb"`
	MemUsedKB       uint64  `json:"mem_used_kb"`
	MemAvailableKB  uint64  `json:"mem_available_kb"`
	NodeIP          string  `json:"node_ip"`
}

// prende le metriche dai watcher
func collectMetrics() ([]Metrics, error) {

	// guarda gli indirizzi ip associati al nome DNS "node-metrics..." restituisce una lista di indirizzi ip
	ips, err := net.LookupHost("node-metrics.default.svc.cluster.local")
	if err != nil {
		return nil, err
	}

	var results []Metrics

	//http.Client è l'oggetto per fare richieste http: il time di 2 secondi dice che se non arriva
	// una risposta in 2 secondi allora fallisce la richiesta
	client := http.Client{Timeout: 2 * time.Second}

	for _, ip := range ips {

		//crei l'url per ogni IP
		url := fmt.Sprintf("http://%s:8080/metrics", ip)
		//fai la richiesta a ogni watcher
		resp, err := client.Get(url)
		if err != nil {
			log.Println("Errore chiamando watcher:", ip, err)
			continue
		}
		//crei variamo m dove salvare le metriche di questo watcher ( poi la aggiungerai alla lista )
		var m Metrics
		//prende il body http (JSON) e lo converte nella struttura metrics
		err = json.NewDecoder(resp.Body).Decode(&m)
		// chiudi lo stream http, sotto cè una connessione TCP aperta che va chiusa
		resp.Body.Close()
		if err != nil {
			log.Println("Errore parsing:", ip, err)
			continue
		}
		// creiamo il risultato
		results = append(results, m)
	}

	return results, nil
}

// struttura dati che rappresenta un pod, così quando otteniamo i json dal proxy
// li possiamo mappare in oggetti Pod
// NB: generalmente un pod ha deti Metadata e il campo Spec
type Pod struct {
	Metadata struct {
		Name string `json:"name"` // struct tag in go: "quando leggi/scrivi json questo campo corrisponde a name e quando ricevi "
		// { "name": "nginx-123" } go lo mette in automatico in pod.Metadata.Name
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels"` // mappa con chiave-valore = string-string
		// se { "labels": { "app": "nginx", "env": "prod"  } } allora
		// Labels["app"] = "nginx" Labels["env"] = "prod"

		OwnerRefs []struct { // OwnerRefs = [{ Kind: "ReplicaSet" }, { Kind: "Deployment" } ]
			Kind string `json:"kind"`
		} `json:"ownerReferences"`
	} `json:"metadata"`

	Spec struct {
		NodeName string `json:"nodeName"` // serve a leggere su quale nodo gira il pod
	} `json:"spec"`
}

// in generale riceveremo dal proxy molti più dati per singolo pod, ma go salverà solo quelli per cui trova uno struct tag compatibile
// scegliamo di salvare solo questi in pratica

// serve anche una struct del genere in quanto l'API non ritorna qualcosa del tipo "[ {...}, {...} ]" bensì
// qualcosa del tipo { "items": [ ... ] }

type PodList struct {
	Items []Pod `json:"items"`
}

// chiama il proxy, si fa dare i pod del namespace default e popola la struttura dati Pod
func getPods() ([]Pod, error) {

	url := proxyURL + "/api/v1/namespaces/default/pods"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var podList PodList
	err = json.NewDecoder(resp.Body).Decode(&podList)

	return podList.Items, err
}

// funzione per eliminare un pod specifico già scelto
func deletePod(name string) {

	// nel kube-api un pod è identificato dal suo nome, quindi costruiamo l'endpoint che punta al pod specifico
	// (passando ovviamente per il proxy)
	url := fmt.Sprintf("%s/api/v1/namespaces/default/pods/%s", proxyURL, name)

	// creiamo una richiesta http manuale ( http.client mette a dispostione solo i metodi post e get )
	req, _ := http.NewRequest("DELETE", url, nil)
	// creo un client http per mandare la richiesta
	client := &http.Client{}

	// mando la richiesta (Do appunto èp geneale )
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Errore DELETE:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Pod eliminato:", name)
}

// per capire se questo pod è sicuro da eliminare
func isEvictable(p Pod) bool {

	// evita pod senza owner (statici)
	if len(p.Metadata.OwnerRefs) == 0 {
		return false
	}

	// evita roba di sistema (extra sicurezza)
	if p.Metadata.Namespace != "default" {
		return false
	}

	return true
}

// loop di riconcilliazione
func reconcile() {

	log.Println("---- RECONCILE ----")

	// prendo le metriche di tutti i nodi
	metrics, err := collectMetrics()
	if err != nil {
		log.Println("Errore metriche:", err)
		return
	}
	// prendo i pod
	pods, err := getPods()
	if err != nil {
		log.Println("Errore pods:", err)
		return
	}

	for _, m := range metrics {

		if m.CPUUsagePercent > 5 {

			log.Println("Nodo in overload:", m.NodeIP)

			for _, p := range pods {

				// evita di eliminare il controller stesso
				if p.Metadata.Labels["app"] == "controller_metrics" {
					continue
				}

				if p.Spec.NodeName == m.NodeIP && isEvictable(p) {

					log.Println("Elimino pod:", p.Metadata.Name)

					deletePod(p.Metadata.Name)

					return // elimino un pod per ciclo
				}
			}
		}
	}
}

func main() {

	log.Println("Controller avviato...")

	for {
		reconcile()
		time.Sleep(10 * time.Second)
	}
}
