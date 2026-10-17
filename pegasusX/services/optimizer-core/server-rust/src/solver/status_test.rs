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
