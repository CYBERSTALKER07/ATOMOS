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

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        match tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate()) {
            Ok(mut signal) => {
                signal.recv().await;
            }
            Err(e) => {
                log::error!("failed to install SIGTERM handler: {}", e);
                std::future::pending::<()>().await;
            }
        }
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("Received Ctrl+C / SIGINT, initiating graceful drain...");
        },
        _ = terminate => {
            info!("Received SIGTERM, initiating graceful drain...");
        },
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    let addr = "0.0.0.0:50055".parse()?;
    let optimizer_service = OptimizerService::default();

    info!("Starting optimizer-core-rust gRPC server on {}", addr);

    Server::builder()
        .add_service(OptimizerCoreServer::new(optimizer_service))
        .serve_with_shutdown(addr, shutdown_signal())
        .await?;

    info!("optimizer-core-rust gRPC server shutdown complete");
    Ok(())
}
