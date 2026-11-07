use crate::optimizer_core::optimizer_core_server::OptimizerCore;
use crate::optimizer_core::{
    OptimizeCpsatRequest, OptimizeCpsatResponse, OptimizeVrpRequest, OptimizeVrpResponse,
};
use log::info;
use std::sync::Arc;
use tokio::sync::Semaphore;
use tonic::{Request, Response, Status};

use crate::solver;

#[derive(Debug, Clone)]
pub struct OptimizerService {
    semaphore: Arc<Semaphore>,
}

impl Default for OptimizerService {
    fn default() -> Self {
        let max_concurrency = std::env::var("OPTIMIZER_MAX_CONCURRENCY")
            .ok()
            .and_then(|v| v.parse::<usize>().ok())
            .unwrap_or(4);
        Self {
            semaphore: Arc::new(Semaphore::new(max_concurrency)),
        }
    }
}

impl OptimizerService {
    pub fn new(max_concurrency: usize) -> Self {
        Self {
            semaphore: Arc::new(Semaphore::new(max_concurrency)),
        }
    }

    pub fn with_semaphore(semaphore: Arc<Semaphore>) -> Self {
        Self { semaphore }
    }
}

#[tonic::async_trait]
impl OptimizerCore for OptimizerService {
    async fn optimize_vrp(
        &self,
        request: Request<OptimizeVrpRequest>,
    ) -> Result<Response<OptimizeVrpResponse>, Status> {
        let req = request.into_inner();
        info!("Received OptimizeVRP request for job_id: {}", req.job_id);

        let permit = self
            .semaphore
            .clone()
            .try_acquire_owned()
            .map_err(|_| Status::resource_exhausted("solver concurrency limit reached"))?;

        let response = tokio::task::spawn_blocking(move || {
            let _permit = permit;
            solver::vrp::solve(req)
        })
        .await
        .map_err(|e| Status::internal(format!("solver task execution failed: {}", e)))?;

        Ok(Response::new(response))
    }

    async fn optimize_cpsat(
        &self,
        request: Request<OptimizeCpsatRequest>,
    ) -> Result<Response<OptimizeCpsatResponse>, Status> {
        let req = request.into_inner();
        info!("Received OptimizeCPSAT request for job_id: {}", req.job_id);

        let permit = self
            .semaphore
            .clone()
            .try_acquire_owned()
            .map_err(|_| Status::resource_exhausted("solver concurrency limit reached"))?;

        let response = tokio::task::spawn_blocking(move || {
            let _permit = permit;
            solver::cpsat::solve(req)
        })
        .await
        .map_err(|e| Status::internal(format!("solver task execution failed: {}", e)))?;

        Ok(Response::new(response))
    }
}
