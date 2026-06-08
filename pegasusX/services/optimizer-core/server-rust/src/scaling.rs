pub const SCALE_FACTOR: f64 = 10000.0;

/// Converts a system float (e.g. kilometers, capacities) into a deterministic solver integer
pub fn to_solver_int(val: f64) -> i64 {
    (val * SCALE_FACTOR).round() as i64
}

/// Converts a solver integer back to system float
pub fn to_system_float(val: i64) -> f64 {
    val as f64 / SCALE_FACTOR
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_to_solver_int() {
        assert_eq!(to_solver_int(1.23456), 12346);
        assert_eq!(to_solver_int(1.23454), 12345);
        assert_eq!(to_solver_int(0.0), 0);
        assert_eq!(to_solver_int(-1.23456), -12346);
    }

    #[test]
    fn test_to_system_float() {
        assert_eq!(to_system_float(12346), 1.2346);
        assert_eq!(to_system_float(0), 0.0);
    }
}
