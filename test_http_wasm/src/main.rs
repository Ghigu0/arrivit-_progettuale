use tokio::time::{timeout, Duration};

#[tokio::main(flavor = "current_thread")]
async fn main() {
    println!("test network");

    let url = "http://10.43.217.22:8080";

    println!("calling {}", url);

    let result = timeout(
        Duration::from_secs(5),
        reqwest::get(url),
    )
    .await;

    match result {
        Ok(Ok(r)) => {
            println!("OK {}", r.status());
        }
        Ok(Err(e)) => {
            eprintln!("ERR {}", e);
        }
        Err(_) => {
            eprintln!("TIMEOUT: richiesta bloccata oltre 5 secondi");
        }
    }
}