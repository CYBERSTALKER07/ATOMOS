import math
import os
from typing import Any, Dict, List, Tuple
from flask import Flask, request, jsonify

app = Flask(__name__)

MODEL_VERSION = "CREDIT_SCORECARD_V2_CALIBRATED"
SCORING_MODE = "CALIBRATED_LOGISTIC_SCORECARD"


def _clamp(val: float, lo: float, hi: float) -> float:
    return max(lo, min(hi, val))


def _evaluate_credit_risk(payload: Dict[str, Any]) -> Tuple[Dict[str, Any], int]:
    retailer_id = str(payload.get("retailer_id") or "").strip()
    if not retailer_id:
        return {"error": "retailer_id is required"}, 400

    try:
        avg_days_to_pay = float(payload.get("avg_days_to_pay", 7.0))
        missed_payments = int(payload.get("missed_payments", 0))
        total_volume_millions = float(payload.get("total_volume_millions", 0.0))
        current_limit_uzs = int(payload.get("current_limit_uzs", 0))
        days_past_due_max = int(payload.get("days_past_due_max", 0))
        active_disputes = int(payload.get("active_disputes", 0))
        completed_orders = int(payload.get("completed_orders", 0))
    except (TypeError, ValueError) as exc:
        return {"error": f"invalid numeric feature: {exc}"}, 400

    if avg_days_to_pay < 0 or missed_payments < 0 or total_volume_millions < 0 or current_limit_uzs < 0:
        return {"error": "credit features must be non-negative"}, 400

    # Calibrated log-odds scorecard for B2B FMCG retailer trade credit (Net-7 / Net-14 / Net-30):
    # Positive log-odds => higher repayment reliability (1 - PD).
    reason_codes: List[str] = []
    log_odds = 0.85  # baseline intercept (~70% reliability for neutral history)

    # 1. Payment velocity penalty/bonus (target Net-7 baseline)
    if avg_days_to_pay <= 5.0:
        log_odds += 0.65
        reason_codes.append("FAST_SETTLEMENT_VELOCITY")
    elif avg_days_to_pay <= 14.0:
        log_odds += 0.20 * ((14.0 - avg_days_to_pay) / 9.0)
    else:
        over_days = min(avg_days_to_pay - 14.0, 60.0)
        log_odds -= 0.065 * over_days
        reason_codes.append("SLOW_SETTLEMENT_VELOCITY")

    # 2. Missed payment / delinquency count penalty (hard impact)
    if missed_payments == 0:
        log_odds += 0.55
        reason_codes.append("ZERO_MISSED_PAYMENTS")
    else:
        log_odds -= 0.85 * min(missed_payments, 10)
        reason_codes.append("MISSED_PAYMENT_HISTORY")

    # 3. Maximum historical days past due (DPD)
    if days_past_due_max > 0:
        log_odds -= 0.04 * min(days_past_due_max, 90)
        if days_past_due_max >= 15:
            reason_codes.append("SEVERE_DAYS_PAST_DUE")

    # 4. Active unresolved disputes
    if active_disputes > 0:
        log_odds -= 0.35 * min(active_disputes, 5)
        reason_codes.append("ACTIVE_DISPUTES_OPEN")

    # 5. Cumulative settled GMV (UZS millions) + order count tenure bonus
    volume_bonus = min(math.log1p(max(total_volume_millions, 0.0)) * 0.32, 1.35)
    log_odds += volume_bonus
    if total_volume_millions >= 10.0:
        reason_codes.append("STRONG_GMV_HISTORY")

    if completed_orders >= 20:
        log_odds += 0.30
        reason_codes.append("ESTABLISHED_ORDER_TENURE")

    repayment_probability = 1.0 / (1.0 + math.exp(-_clamp(log_odds, -8.0, 8.0)))
    default_probability = 1.0 - repayment_probability

    if repayment_probability >= 0.82 and missed_payments == 0 and days_past_due_max <= 7:
        risk_tier = "LOW"
    elif repayment_probability >= 0.65 and missed_payments <= 1:
        risk_tier = "MEDIUM"
    elif repayment_probability >= 0.40:
        risk_tier = "HIGH"
    else:
        risk_tier = "CRITICAL"

    eligible_for_increase = (
        risk_tier in ("LOW", "MEDIUM")
        and repayment_probability >= 0.72
        and missed_payments == 0
        and total_volume_millions >= 2.0
    )

    suggested_increase_uzs = 0
    if eligible_for_increase and current_limit_uzs > 0:
        multiplier = 0.25 if risk_tier == "LOW" else 0.12
        # Cap single step increase to 30% of historical volume in UZS minor-equivalent
        volume_cap_uzs = int(total_volume_millions * 1_000_000 * 0.30)
        raw_increase = int(round(current_limit_uzs * multiplier))
        suggested_increase_uzs = min(raw_increase, max(volume_cap_uzs, raw_increase))

    return {
        "retailer_id": retailer_id,
        "eligible_for_increase": bool(eligible_for_increase),
        "confidence_score": round(float(repayment_probability), 4),
        "default_probability": round(float(default_probability), 4),
        "risk_tier": risk_tier,
        "suggested_increase_uzs": int(suggested_increase_uzs),
        "recommended_limit_uzs": int(current_limit_uzs + suggested_increase_uzs),
        "reason_codes": reason_codes,
        "model_version": MODEL_VERSION,
        "scoring_mode": SCORING_MODE,
    }, 200


@app.route("/healthz", methods=["GET"])
@app.route("/readyz", methods=["GET"])
def healthz():
    return jsonify({
        "status": "ok",
        "service": "ml-scorer-py",
        "model_version": MODEL_VERSION,
        "scoring_mode": SCORING_MODE,
    }), 200


@app.route("/credit/score", methods=["POST"])
def score_credit():
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        return jsonify({"error": "JSON object body required"}), 400
    body, status = _evaluate_credit_risk(data)
    return jsonify(body), status


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8081))
    app.run(host="0.0.0.0", port=port)

