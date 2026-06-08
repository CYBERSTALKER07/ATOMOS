pub mod optimizer_core {
    tonic::include_proto!("optimizer_core");
}

pub mod scaling;
pub mod service;
pub mod solver;

use log::info;
use optimizer_core::optimizer_core_server::OptimizerCoreServer;
use service::OptimizerService;
use tonic::transport::Server;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    let addr = "0.0.0.0:50055".parse()?;
    let optimizer_service = OptimizerService::default();

    info!("Starting optimizer-core-rust gRPC server on {}", addr);

    Server::builder()
        .add_service(OptimizerCoreServer::new(optimizer_service))
        .serve(addr)
        .await?;

    Ok(())
}
