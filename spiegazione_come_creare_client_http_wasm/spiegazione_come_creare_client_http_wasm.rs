
// questa macro qua permette di scrivere asyn prima della funzione main, e permette quindi di usare la funzione .await
// current_thread dice che useremo un thread (come vuole wasm)
#[tokio::main(flavor = "current_thread")]
// in Rust, result < T, E> specifica il tipo di ritorno con T successo E errore
// in questo caso abbiamo () come T, quindi non riporta nulla 
// Result si usa solo quando una funzione può fallire

async fn main() -> Result<(), reqwest::Error> {


    let url = "http://eu.httpbin.org/get?msg=WasmEdge";

    eprintln!("Fetching {:?}...", url);

    // traduzione: reqwest è la libreria importata, mentre :: permette di accedere alle funzioni per accedere a tale libreia
    // come fare in go http.get()
    /*  reqwest::get(url)
            ↓
        crea richiesta HTTP
            ↓
        .await
            ↓
        aspetta risposta
            ↓
        ?
            ↓
        gestisce errore
            ↓
        res = Response*/

    let res = reqwest::get(url).await?;
    // le future sono, ok, prepariamo questa funzione da eseguire ma non la eseguiamo. la await la esegue in modo asincrono
    /* Cosa restituisce .await
    reqwest::get(url).await

    NON restituisce direttamente la risposta ❗
    restituisce:

    Result<Response, reqwest::Error>
    Quindi hai
    Ok(Response)   // successo
    Err(Error)     // errore
    Cosa fa ?

    il ? dice:

    se è Ok → prendi il valore
    se è Err → esci dalla funzione
     Traduzione esplicita
    let res = reqwest::get(url).await?;

    ≈

    let res = match reqwest::get(url).await {
        Ok(v) => v,
        Err(e) => return Err(e),
    };*/
   
    eprintln!("Response: {:?} {}", res.version(), res.status());
    eprintln!("Headers: {:#?}\n", res.headers());

    /* in res non hai mica il risultato di tutta la richieste, nono
        hai solo "aperto" la connessione 
        Dentro Response hai:

        status (200 OK)
        headers
        connessione aperta
        body già letto

        quindi il body NON è ancora stato letto, perché il body arriva come uno stream (flusso di dati)

        res.text() quindi prende tutti i chunk di dati e li mette in una stringa
    */
    let body = res.text().await?;
    // non è che è asincrona così esegue la println dopo, è asincrona così il runtime tokio fa altre cose sotto
    println!("GET: {}", body);



    // invece per mandare delle richieste post dobbiamo un attimo dichiarare il client in modo esplicito 
    // ( anche con la get si creava, ma in modo implicito)

    let client = reqwest::Client::new();

    // abbastanza autoesplicativo
    let res = client
        .post("http://eu.httpbin.org/post")
        .body("msg=WasmEdge")
        .send()
        .await?;
    let body = res.text().await?;

    println!("POST: {}", body);


    // autoesplicativo
    let res = client
        .put("http://eu.httpbin.org/put")
        .body("msg=WasmEdge")
        .send()
        .await?;
    let body = res.text().await?;

    println!("PUT: {}", body);

    Ok(())
}

// aggiungi su ipad che non usiamo dns come in collector normale ma usiamo il proxy che chiama l'api server
//scaletta del mio progetto
/*GET endpoints → lista IP watcher
GET metrics da ogni watcher
GET pods
identifica nodi overload
DELETE pod*/


/* file cargo toml da usare 




[package]
name = "wasmedge_reqwest_demo"
version = "0.1.0"
edition = "2021"

[patch.crates-io]
tokio = { git = "https://github.com/second-state/wasi_tokio.git", branch = "v1.40.x" }
mio = { git = "https://github.com/second-state/wasi_mio.git", branch = "v1.0.x" }
socket2 = { git = "https://github.com/second-state/socket2.git", branch = "v0.5.x" }
hyper = { git = "https://github.com/second-state/wasi_hyper.git", branch = "v0.14.x" }
reqwest = { git = "https://github.com/second-state/wasi_reqwest.git", branch = "0.11.x" }

[dependencies]
reqwest = { version = "0.11", default-features = false, features = ["rustls-tls"] }
tokio = { version = "1", features = ["rt", "macros", "net", "time"] }
tokio-util = { version = ">=0.7.1, <0.7.18" }




  */