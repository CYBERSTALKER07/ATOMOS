use crate::optimizer_core::optimizer_core_server::OptimizerCore;
use crate::optimizer_core::{OptimizeVrpRequest, OptimizeVrpResponse, OptimizeCpsatRequest, OptimizeCpsatResponse};
use tonic::{Request, Response, Status};
use log::info;

use crate::solver;

#[derive(Debug, Default)]
pub struct OptimizerService {}

#[tonic::async_trait]
impl OptimizerCore for OptimizerService {
    async fn optimize_vrp(
        &self,
        request: Request<OptimizeVrpRequest>,
    ) -> Result<Response<OptimizeVrpResponse>, Status> {
        let req = request.into_inner();
        info!("Received OptimizeVRP request for job_id: {}", req.job_id);

        let response = solver::vrp::solve(req);

        Ok(Response::new(response))
    }

    async fn optimize_cpsat(
        &self,
        request: Request<OptimizeCpsatRequest>,
    ) -> Result<Response<OptimizeCpsatResponse>, Status> {
        let req = request.into_inner();
        info!("Received OptimizeCPSAT request for job_id: {}", req.job_id);

        let response = solver::cpsat::solve(req);

        Ok(Response::new(response))
    }
}
