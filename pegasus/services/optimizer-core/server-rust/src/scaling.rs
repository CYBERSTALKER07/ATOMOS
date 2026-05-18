#![allow(dead_code)]

pub const SCALE_FACTOR: f64 = 10_000.0;

pub fn to_solver_int(value: f64) -> i64 {
    (value * SCALE_FACTOR).round() as i64
}

pub fn to_system_float(value: i64) -> f64 {
    value as f64 / SCALE_FACTOR
}

pub fn scale_slice(values: &[f64]) -> Vec<i64> {
    values.iter().map(|value| to_solver_int(*value)).collect()
}

pub fn scale_matrix(values: &[Vec<f64>]) -> Vec<Vec<i64>> {
    values.iter().map(|row| scale_slice(row)).collect()
}

#[cfg(test)]
mod tests {
    use super::{to_solver_int, to_system_float};

    #[test]
    fn converts_between_system_and_solver_units() {
        let samples = [0.0_f64, 1.2345_f64, 9.9999_f64, -3.25_f64];
        for sample in samples {
            let scaled = to_solver_int(sample);
            let roundtrip = to_system_float(scaled);
            assert!(
                (sample - roundtrip).abs() <= 0.0001,
                "sample={sample}, roundtrip={roundtrip}"
            );
        }
    }

    #[test]
    fn follows_expected_golden_vectors() {
        assert_eq!(to_solver_int(0.0), 0);
        assert_eq!(to_solver_int(1.0), 10_000);
        assert_eq!(to_solver_int(1.2345), 12_345);
        assert_eq!(to_solver_int(9.87654), 98_765);
    }
}
