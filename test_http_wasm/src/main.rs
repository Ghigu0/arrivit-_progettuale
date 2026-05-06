#[tokio::main(flavor = "current_thread")]
async fn main() {
    println!("test network");
    match reqwest::get("http://10.96.0.1:443").await {
        Ok(r) => println!("OK {}", r.status()),
        Err(e) => eprintln!("ERR {}", e),
    }
}
