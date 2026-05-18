use tonic::{Request, Response, Status};

use crate::pb::{
    optimizer_core_service_server::OptimizerCoreService as OptimizerCoreServiceTrait, CpsatRequest,
    CpsatResponse, HealthCheckRequest, HealthCheckResponse, VrpRequest, VrpResponse,
};

#[derive(Default)]
pub struct OptimizerCoreService;

#[tonic::async_trait]
impl OptimizerCoreServiceTrait for OptimizerCoreService {
    async fn calculate_route(
        &self,
        request: Request<VrpRequest>,
    ) -> Result<Response<VrpResponse>, Status> {
        let request = request.into_inner();
        let response = crate::solver::vrp::solve(&request);
        Ok(Response::new(response))
    }

    async fn resolve_constraint(
        &self,
        request: Request<CpsatRequest>,
    ) -> Result<Response<CpsatResponse>, Status> {
        let request = request.into_inner();
        let response = crate::solver::cpsat::solve(&request);
        Ok(Response::new(response))
    }

    async fn health_check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> Result<Response<HealthCheckResponse>, Status> {
        Ok(Response::new(HealthCheckResponse {
            ok: true,
            service: "optimizer-core-rust".to_string(),
        }))
    }
}
