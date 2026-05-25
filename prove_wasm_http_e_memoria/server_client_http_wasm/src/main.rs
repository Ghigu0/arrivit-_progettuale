use std::convert::Infallible;

use hyper::body::Body;
use hyper::header::CONTENT_TYPE;
use hyper::service::{make_service_fn, service_fn};
use hyper::{Method, Request, Response, Server, StatusCode};

use tokio::net::TcpListener;
use tokio::time::{timeout, Duration};

const PROXY_URL: &str = "http://10.43.34.218:8080/api/v1/namespaces/default/pods";

#[tokio::main(flavor = "current_thread")]
async fn main() {
    println!("wasm http server starting");
    println!("proxy url: {}", PROXY_URL);

    let addr = "0.0.0.0:8080";

    let tcp_listener = TcpListener::bind(addr)
        .await
        .expect("failed to bind tcp listener");

    println!("listening on {}", addr);

    let std_listener = tcp_listener
        .into_std()
        .expect("failed to convert tokio listener into std listener");

    let make_svc = make_service_fn(|_conn| async {
        Ok::<_, Infallible>(service_fn(handle_request))
    });

    Server::from_tcp(std_listener)
        .expect("failed to create hyper server from tcp listener")
        .serve(make_svc)
        .await
        .expect("server error");
}

async fn handle_request(req: Request<Body>) -> Result<Response<Body>, Infallible> {
    println!("incoming request: {} {}", req.method(), req.uri().path());

    let response = match (req.method(), req.uri().path()) {
        (&Method::GET, "/health") => json_response(
            StatusCode::OK,
            r#"{"status":"ok"}"#.to_string(),
        ),

        (&Method::GET, "/test/pods") => call_proxy().await,

        _ => json_response(
            StatusCode::NOT_FOUND,
            r#"{"error":"not found"}"#.to_string(),
        ),
    };

    Ok(response)
}

async fn call_proxy() -> Response<Body> {
    println!("calling {}", PROXY_URL);

    let result = timeout(
        Duration::from_secs(5),
        reqwest::get(PROXY_URL),
    )
    .await;

    match result {
        Ok(Ok(resp)) => {
            let status = resp.status();

            println!("OK {}", status);

            json_response(
                StatusCode::OK,
                format!(
                    r#"{{"status":"ok","upstream_status":{}}}"#,
                    status.as_u16()
                ),
            )
        }

        Ok(Err(e)) => {
            eprintln!("ERR {}", e);

            json_response(
                StatusCode::BAD_GATEWAY,
                format!(
                    r#"{{"status":"error","message":"{}"}}"#,
                    escape_json(&e.to_string())
                ),
            )
        }

        Err(_) => {
            eprintln!("TIMEOUT");

            json_response(
                StatusCode::GATEWAY_TIMEOUT,
                r#"{"status":"timeout"}"#.to_string(),
            )
        }
    }
}

fn json_response(status: StatusCode, body: String) -> Response<Body> {
    Response::builder()
        .status(status)
        .header(CONTENT_TYPE, "application/json")
        .body(Body::from(body))
        .unwrap()
}

fn escape_json(s: &str) -> String {
    s.replace('\\', "\\\\").replace('"', "\\\"")
}