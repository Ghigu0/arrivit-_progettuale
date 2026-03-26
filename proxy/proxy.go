package main

import (
	"crypto/tls"	// per gestire connessioni https con tls (per comunicare al kube-API dobbiamo usare http)
	"crypto/x509"	// serve per lavorare con i certificati CA
	"io"			// per leggere e scrivere stream di dati
	"log"			// serve per stampare i log (come una stampa normale ma aggiunge il timestamp)
	"net/http"		// libreria che serve per creare server http e fare richieste http
	"os"			// serve per interagire con il file system 
)

/*
	- il token path serve per autenticarsi verso l'API server
	- il certificato serve per assicurarsi che stai parlando con kubernetes (per evitare attacchi MITM)

*/
const (
	apiServer = "https://kubernetes.default.svc"						// indirizzo interno del server API di kubernetes 
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"	// percorso del token di autenticazione (montato in automatico da kubernetes)
	caPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"	// percorso del certificato (montato in automatico da kubernetes)
)

// main, punto di ingresso del programma
func main() {

	client := createHTTPClient()										// il client è un oggetto che serve per mandare richieste http/https verso altri server

																		// registra solo cosa fare quando arriveranno delle richiest al proxy (non hai ancora creato "il server",è la prossima riga di codice )
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) 
	{ handleProxy(client, w, r)})
																		// serve per registrare un handler http, ovvero dire al server cosa fare quando arriva una richiesta 
																		// "/" è un pattern, significa che verrano intercettate tutte le richieste tipo /, /api, /pods etc
																		// func è una funzione inline (anonima) significa che viene eseguita ogni volta che arriva una richiesta
																		// handleFunc vuole dentro una funzione con la firma "func(w http.ResponseWriter, r *http.Request)", per questo non puoi mettere direttament handleProxy
	
																		// dentro al package net//http esiste uno stato globale, ecco perchè con la funzione http.HandleFunc puoi settare un handler e in un secondo momento avviare i lserver 
	log.Println("Proxy listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))						// il secondo paramentro è un handler, qui messo nil, si usa quello de dafault (quindi quello fatto prima )
}

func createHTTPClient() *http.Client { 									// restituisce un puntatore a http.Client
	
	caCert, err := os.ReadFile(caPath)									// leggi il certificato CA dal filesystem. la variabile err contiene eventuali errori
	if err != nil {
		log.Fatalf("Errore lettura CA: %v", err)						//Fatalf termina il programma 
	}

	caCertPool := x509.NewCertPool()									// crea un contenitore di certificati trusted 
	caCertPool.AppendCertsFromPEM(caCert)								// aggiunge al pool il certificato

	
	tlsConfig := &tls.Config{											// tls.Config è una struttura che dice al client come comportarsi in caso di connessioni https
		RootCAs: caCertPool,													//in questo caso in questa struttura ci mettiamo il "pool" di certificati (è uno solo)
	}																	// serve solo a configurare TLS, "non sa niente di http" per questo serve poi tr			

	tr := &http.Transport{												// per le comunicazioni http in generale. in questo tipo di comunicazione ci emttiamo la struttura per le https
		TLSClientConfig: tlsConfig,												// gli diciamo come comportarsi in caso di https 
	}

	return &http.Client{Transport: tr}									// crea e ritorna il client
}

// funzione per leggere il token dal file system
func getToken() string {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		log.Fatalf("Errore lettura token: %v", err)
	}
	return string(token)
}


// parametri client -> per fare richieste a Kubernetes, w -> per rispondere al client, r -> richiesta ricevuta
func handleProxy(client *http.Client, w http.ResponseWriter, r *http.Request) {
	
	token := getToken()													// leggo il token 

																		// Costruisci URL verso API server
	targetURL := apiServer + r.URL.Path									// se per esempio r.URL.Path = "/api/v1/pods" allora targetURL = "https://kubernetes.default.svc" + "/api/v1/pods"
	if r.URL.RawQuery != "" {											// una URL è fatta così /path?chiave=valore&chiave2=valore2. r.URL.path è solo la prima parte /path, mentre r.URL.RawQuery è la parte dopo il ?
		targetURL += "?" + r.URL.RawQuery								// insomma ci concateno i parametri 
	}

	// req: variabile della risposta
	req, err := http.NewRequest(r.Method, targetURL, r.Body)			// con questa funzione crei una richiesta che il tuo server deve mandare, come vedi
	if err != nil {															// dai parametri, riusi il metodo e il body della richiesta originale, ma metti io nuovo target URL ( quello di kubernetes)
		http.Error(w, "Errore creazione richiesta", 500)
		return
	}

	// aggiungiamo a req gli headers originali							// add per aggiungere più valore senza cancellarne altri
	for key, values := range r.Header {									// es: Add("Accept", "application/json") A
		for _, v := range values {										// 	   Add("Accept", "text/plain")
			req.Header.Add(key, v)
		}
	}

	// Aggiungi autenticazione
	req.Header.Set("Authorization", "Bearer "+token)					// set sovrascrive tutto. attenzione, lo standard http è proprio un header con formato "Authorization: Bearer <token>" quindi attenzione ai typo








// sono arrivato quaaa 




	// Content-Type fallback
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Errore chiamata API Kubernetes", 500)
		return
	}
	defer resp.Body.Close()

	// Copia status code
	w.WriteHeader(resp.StatusCode)

	// Copia headers risposta
	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	// Copia body
	io.Copy(w, resp.Body)
}



/*  A COSA SERVE ESATTAMENTE IL CERTIFICATO CA ? 

	i certificati servono per le connessioni https.
	normalmente nel web, un sito crea un certificato che viene firmato da una 
	autorità fidata (un ente terzo) che ti viene poi mandato prima di instaurare 
	la connessione https, proprio perchè contiene la chiave pubblica che serve 
	per cifrare i dati verso il server (poi il server per decifrarli usa la chiave 
	privata) che poi in realtà serve per generare la chiave segreta con cui poi 
	effettivamente si comunica in modo cifrato

	1. ricevi certificato (pubblico)
	2. verifichi CA
	3. fai handshake
	4. generi chiave segreta
	5. comunichi cifrato
	
*/

/*  A COSA SERVE ESATTAMENTE IL TOKEN ? 

	il token serve per autenticarsi verso l'api di kubernetes. non ha a che fare 
	con tls e https, quello è solo per comunicare, serve proprio perchè l'api 
	riceve la tua richiesta e dice "ah si, questo token è valido, allora mi fido
	di questo componente e gli do quello che vuole" (es la lista di pod) il token
	è un JWT difatto. dopo che con il token verifica chi sei, guarderà i tuoi 
	permessi RBAC ( che andremo a settare con un manifest a parte ) per vedere se
	effettivamente hai i permessi per, per esempio, ottenere l'elenco dei pod etc

*/