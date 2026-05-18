mod mapping;
mod scaling;
mod service;
mod solver;

pub mod pb {
    tonic::include_proto!("pegasus.optimizer.core.v1");
}

use std::env;
use std::net::SocketAddr;

use tonic::transport::Server;
use tracing_subscriber::EnvFilter;

use crate::pb::optimizer_core_service_server::OptimizerCoreServiceServer;
use crate::service::OptimizerCoreService;

fn env_u16(name: &str, fallback: u16) -> u16 {
    env::var(name)
        .ok()
        .and_then(|raw| raw.parse::<u16>().ok())
        .unwrap_or(fallback)
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .with_target(false)
        .compact()
        .init();

    let port = env_u16("OPTIMIZER_CORE_GRPC_PORT", 50055);
    let address: SocketAddr = format!("0.0.0.0:{port}").parse()?;

    tracing::info!(port, "optimizer-core-rust gRPC server starting");

    Server::builder()
        .add_service(OptimizerCoreServiceServer::new(
            OptimizerCoreService::default(),
        ))
        .serve(address)
        .await?;

    Ok(())
}
