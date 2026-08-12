use crate::optimizer_core::{OptimizeVrpRequest, SolverStatus, VrpNodeDemand, VrpVehicle};
use crate::solver::vrp;

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
