use crate::optimizer_core::{
    CpsatFactorySlot, CpsatManifestRequirement, OptimizeCpsatRequest, OptimizeVrpRequest,
    SolverStatus, VrpNodeDemand, VrpVehicle,
};
use crate::solver::{cpsat, vrp};

#[test]
fn greedy_vrp_never_reports_optimal() {
    let req = OptimizeVrpRequest {
        job_id: "t1".into(),
        depot_node_uuid: "depot".into(),
        drop_off_node_uuids: vec!["n1".into()],
        // 2x2 matrix: depot↔n1
        distance_matrix_km: vec![0.0, 1.0, 1.0, 0.0],
        matrix_size: 2,
        vehicles: vec![VrpVehicle {
            vehicle_uuid: "v1".into(),
            driver_uuid: "d1".into(),
            capacity_vu: 100.0,
            start_window_hours: 0.0,
            end_window_hours: 24.0,
        }],
        node_demands: vec![VrpNodeDemand {
            node_uuid: "n1".into(),
            demand_vu: 1.0,
        }],
        node_time_windows: vec![],
        solver_time_limit_ms: 1000,
    };
    let resp = vrp::solve(req);
    assert_ne!(
        resp.status,
        SolverStatus::Optimal as i32,
        "greedy must not claim OPTIMAL"
    );
    assert_eq!(resp.status, SolverStatus::Heuristic as i32);
}

/// G6.C: legacy CP_SAT factory assign is greedy — never OPTIMAL.
#[test]
fn greedy_cpsat_never_reports_optimal() {
    let req = OptimizeCpsatRequest {
        job_id: "cpsat-t1".into(),
        factory_slots: vec![CpsatFactorySlot {
            factory_node_uuid: "f1".into(),
            slot_capacity: 100.0,
        }],
        manifest_requirements: vec![CpsatManifestRequirement {
            manifest_id: "m1".into(),
            required_capacity: 10.0,
            priority_score: 5.0,
            eligible_factory_node_uuids: vec!["f1".into()],
        }],
        solver_time_limit_ms: 1000,
        num_search_workers: 1,
    };
    let resp = cpsat::solve(req);
    assert_ne!(
        resp.status,
        SolverStatus::Optimal as i32,
        "greedy CPSAT path must not claim OPTIMAL"
    );
    assert_eq!(resp.status, SolverStatus::Heuristic as i32);
}

#[test]
fn invalid_distance_matrix_dimensions_returns_structured_error() {
    let req = OptimizeVrpRequest {
        job_id: "dim-err".into(),
        depot_node_uuid: "depot".into(),
        drop_off_node_uuids: vec!["n1".into()],
        // 2x2 matrix expected (4 elements), but provide only 3
        distance_matrix_km: vec![0.0, 1.0, 1.0],
        matrix_size: 2,
        vehicles: vec![VrpVehicle {
            vehicle_uuid: "v1".into(),
            driver_uuid: "d1".into(),
            capacity_vu: 100.0,
            start_window_hours: 0.0,
            end_window_hours: 24.0,
        }],
        node_demands: vec![VrpNodeDemand {
            node_uuid: "n1".into(),
            demand_vu: 1.0,
        }],
        node_time_windows: vec![],
        solver_time_limit_ms: 1000,
    };
    let resp = vrp::solve(req);
    assert_eq!(
        resp.status,
        SolverStatus::ModelInvalid as i32,
        "dimension mismatch must return MODEL_INVALID"
    );
    assert!(!resp.warnings.is_empty(), "warnings must not be empty");

    let has_structured_warning = resp.warnings.iter().any(|w| {
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(w) {
            v["code"] == "INVALID_DISTANCE_MATRIX_DIMENSIONS"
                && v["expected_elements"] == 4
                && v["received_elements"] == 3
                && v["nodes_count"] == 2
        } else {
            false
        }
    });
    assert!(
        has_structured_warning,
        "expected structured warning with INVALID_DISTANCE_MATRIX_DIMENSIONS, expected=4, received=3, got {:?}",
        resp.warnings
    );
}

#[test]
fn solver_time_limit_negative_and_zero_sanitized() {
    // Negative or zero time limit must default to 2_000ms instead of wrapping to u64::MAX
    let req = OptimizeVrpRequest {
        job_id: "neg-time".into(),
        depot_node_uuid: "depot".into(),
        drop_off_node_uuids: vec!["n1".into()],
        distance_matrix_km: vec![0.0, 1.0, 1.0, 0.0],
        matrix_size: 2,
        vehicles: vec![VrpVehicle {
            vehicle_uuid: "v1".into(),
            driver_uuid: "d1".into(),
            capacity_vu: 100.0,
            start_window_hours: 0.0,
            end_window_hours: 24.0,
        }],
        node_demands: vec![],
        node_time_windows: vec![],
        solver_time_limit_ms: -100,
    };
    let resp = vrp::solve(req);
    assert_eq!(resp.status, SolverStatus::Heuristic as i32);

    let cpsat_req = OptimizeCpsatRequest {
        job_id: "cpsat-neg".into(),
        factory_slots: vec![],
        manifest_requirements: vec![],
        solver_time_limit_ms: -50,
        num_search_workers: 1,
    };
    let cpsat_resp = cpsat::solve(cpsat_req);
    assert_eq!(cpsat_resp.status, SolverStatus::Heuristic as i32);
}

#[tokio::test]
async fn optimizer_service_concurrency_bound_backpressure() {
    use crate::optimizer_core::optimizer_core_server::OptimizerCore;
    use crate::service::OptimizerService;
    use tonic::Code;

    // Create service with semaphore capacity = 1
    let service = OptimizerService::new(1);

    // Acquire the only permit
    let req = OptimizeVrpRequest {
        job_id: "t-conc".into(),
        depot_node_uuid: "depot".into(),
        drop_off_node_uuids: vec!["n1".into()],
        distance_matrix_km: vec![0.0, 1.0, 1.0, 0.0],
        matrix_size: 2,
        vehicles: vec![VrpVehicle {
            vehicle_uuid: "v1".into(),
            driver_uuid: "d1".into(),
            capacity_vu: 100.0,
            start_window_hours: 0.0,
            end_window_hours: 24.0,
        }],
        node_demands: vec![],
        node_time_windows: vec![],
        solver_time_limit_ms: 100,
    };

    // First request should succeed
    let resp1 = service.optimize_vrp(tonic::Request::new(req.clone())).await;
    assert!(resp1.is_ok());

    // When we hold all permits, next request must return ResourceExhausted
    let sem = std::sync::Arc::new(tokio::sync::Semaphore::new(1));
    let held_permit = sem.clone().try_acquire_owned().unwrap();
    let saturated_service = OptimizerService::with_semaphore(sem);

    let err = saturated_service
        .optimize_vrp(tonic::Request::new(req))
        .await
        .unwrap_err();
    assert_eq!(err.code(), Code::ResourceExhausted);
    assert!(err.message().contains("solver concurrency limit reached"));

    drop(held_permit);
}
