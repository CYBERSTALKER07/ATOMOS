from __future__ import annotations

SCALE_FACTOR = 10_000


def scale_float_to_int(value: float, factor: int = SCALE_FACTOR) -> int:
    return int(round(value * factor))


def unscale_int_to_float(value: int, factor: int = SCALE_FACTOR) -> float:
    return float(value) / float(factor)


def scale_list(values: list[float], factor: int = SCALE_FACTOR) -> list[int]:
    return [scale_float_to_int(v, factor) for v in values]


def scale_matrix(values: list[list[float]], factor: int = SCALE_FACTOR) -> list[list[int]]:
    return [[scale_float_to_int(cell, factor) for cell in row] for row in values]
