use serde::Deserialize;
use tokio::time::{sleep, Duration};

// ======================= STRUCT =======================

// la direttiva sottostante genera codice per convertire automaticamente JSON → struct
#[derive(Deserialize, Debug)]
pub struct Metrics {
    pub cpu_usage_percent: f64,
    pub mem_total_kb: u64,
    pub mem_used_kb: u64,
    pub mem_available_kb: u64,
    pub node_name: String,
}

// risposta endpoint node-metric (gli ip )
#[derive(Deserialize, Debug)]
pub struct Endpoints {
    pub subsets: Vec<Subset>,
}

#[derive(Deserialize, Debug)]
pub struct Subset {
    pub addresses: Option<Vec<Address>>,
}

#[derive(Deserialize, Debug)]
pub struct Address {
    pub ip: String,
}

// risposta endpoint pod
#[derive(Deserialize, Debug)]
pub struct PodList {
    pub items: Vec<Pod>,
}

#[derive(Deserialize, Debug)]
pub struct Pod {
    pub metadata: Metadata,
    pub spec: Spec,
}

#[derive(Deserialize, Debug)]
pub struct Metadata {
    pub name: String,
    pub namespace: String,
    pub labels: Option<std::collections::HashMap<String, String>>,

    #[serde(rename = "ownerReferences")]
    pub owner_references: Option<Vec<OwnerReference>>,
}

#[derive(Deserialize, Debug)]
pub struct OwnerReference {
    pub kind: String,
}

#[derive(Deserialize, Debug)]
pub struct Spec {
    #[serde(rename = "nodeName")]
    pub node_name: String,
}

// ======================= RECONCILE =======================

async fn reconcile() {

    let url_pods = "http://proxy-service:8080/api/v1/namespaces/default/pods";
    /* ritorna qualcosa del tipo
        {
            "items": [
                {
                "metadata": {
                    "name": "nginx-123",
                    "namespace": "default"
                },
                "spec": {
                    "nodeName": "worker-1"
                }
                }
            ]
        }
    */

    let url_watchers = "http://proxy-service:8080/api/v1/namespaces/default/endpoints/node-metrics";
    /* ritorna qualcosa tipo
        {
            "subsets": [
                {
                "addresses": [
                    { "ip": "10.244.0.12" },
                    { "ip": "10.244.0.13" }
                ],
                "ports": [
                    { "port": 8080 }
                ]
                }
            ]
        }
    */

    println!("--- RECONCILE ---");

    //#################################################################### prima parte: ottengo gli ip dei watcher
    // {:?} segnaposto per url_watchers
    eprintln!("Fetching {:?}...", url_watchers);

    let res_ip = match reqwest::get(url_watchers).await {
        Ok(r) => r,
        Err(e) => {
            eprintln!("Errore endpoints: {}", e);
            return;
        }
    };

    eprintln!("Response: {:?} {}", res_ip.version(), res_ip.status());
    eprintln!("Headers: {:#?}\n", res_ip.headers());

    let endpoints: Endpoints = match res_ip.json().await {
        Ok(e) => e,
        Err(e) => {
            eprintln!("Errore parsing endpoints: {}", e);
            return;
        }
    };

    println!("Risultato della get degli ip dei watcher: {:?}", endpoints);

    // vettore finale di IP
    let mut ips: Vec<String> = Vec::new();

    // estrazione IP
    for subset in endpoints.subsets {
        if let Some(addresses) = subset.addresses {
            for addr in addresses {
                ips.push(addr.ip);
            }
        }
    }

    // stampa per debug
    println!("IP watcher: {:?}", ips);

    //#################################################################### seconda parte: interrogo i watcher 

    // in rust le variabili sono immutabili, con mut le puoi modificare
    let mut all_metrics: Vec<Metrics> = Vec::new();

    for ip in ips {
        // creo l'url ogni volta
        let url = format!("http://{}:8080/metrics", ip);

        eprintln!("Chiamo watcher: {}", url);

        // senza ? perchè non voglio uscire dalla funzione
        let res = match reqwest::get(&url).await {
            Ok(r) => r,
            Err(e) => {
                eprintln!("Errore chiamando {}: {}", ip, e);
                continue;
            }
        };

        let metric = match res.json::<Metrics>().await {
            Ok(m) => m,
            Err(e) => {
                eprintln!("Errore parsing {}: {}", ip, e);
                continue;
            }
        };

        all_metrics.push(metric);
    }

    println!("Tutte le metriche: {:?}", all_metrics);

    //#################################################################### terza parte: interrogo i pod 

    eprintln!("Fetching pods: {}", url_pods);

    let res_pods = match reqwest::get(url_pods).await {
        Ok(r) => r,
        Err(e) => {
            eprintln!("Errore chiamando API pods: {}", e);
            return;
        }
    };

    eprintln!("Response: {:?} {}", res_pods.version(), res_pods.status());
    eprintln!("Headers: {:#?}\n", res_pods.headers());

    let pod_list = match res_pods.json::<PodList>().await {
        Ok(p) => p,
        Err(e) => {
            eprintln!("Errore parsing pods: {}", e);
            return;
        }
    };

    let pods: Vec<Pod> = pod_list.items;

    println!("Pods: {:?}", pods);

    //#################################################################### quarta parte: logica (overload → delete)

    for m in &all_metrics {

        if m.cpu_usage_percent > 5.0 {

            println!("Nodo in overload: {}", m.node_name);

            for p in &pods {

                // evita di eliminare il controller stesso
                if let Some(labels) = &p.metadata.labels {
                    if let Some(app) = labels.get("app") {
                        if app == "controller_metrics" {
                            continue;
                        }
                    }
                }

                println!("Controllo pod: {}", p.metadata.name);
                println!(
                    "spec.nodeName={} vs metric.node_name={}",
                    p.spec.node_name, m.node_name
                );

                if p.spec.node_name == m.node_name && is_evictable(p) {

                    println!("Elimino pod: {}", p.metadata.name);

                    delete_pod(&p.metadata.name).await;

                    return; // elimino un pod per ciclo
                }
            }
        }
    }
}

// ======================= HELPERS =======================

// per capire se questo pod è sicuro da eliminare
fn is_evictable(p: &Pod) -> bool {

    // evita pod senza owner (statici)
    if let Some(owners) = &p.metadata.owner_references {
        if owners.is_empty() {
            return false;
        }
    } else {
        return false;
    }

    // evita roba fuori namespace default
    if p.metadata.namespace != "default" {
        return false;
    }

    true
}

// funzione per eliminare un pod specifico
async fn delete_pod(name: &str) {

    let url = format!(
        "http://proxy-service:8080/api/v1/namespaces/default/pods/{}",
        name
    );

    let client = reqwest::Client::new();

    let res = client.delete(&url).send().await;

    match res {
        Ok(_) => println!("Pod eliminato: {}", name),
        Err(e) => eprintln!("Errore DELETE {}: {}", name, e),
    }
}

// ======================= MAIN LOOP =======================

// aggiungi su ipad che non usiamo dns ma usiamo il proxy verso api server
// scaletta:
// GET endpoints → lista IP watcher
// GET metrics da ogni watcher
// GET pods
// identifica nodi overload
// DELETE pod

#[tokio::main(flavor = "current_thread")]
async fn main() {

    println!("Controller avviato...");

    loop {
        reconcile().await;
        sleep(Duration::from_secs(10)).await;
    }
}