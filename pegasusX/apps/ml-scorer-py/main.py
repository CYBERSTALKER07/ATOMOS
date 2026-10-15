import os
from flask import Flask, request, jsonify
import pandas as pd
from sklearn.ensemble import RandomForestClassifier

app = Flask(__name__)

# Mock pre-trained model for demonstration
# In production, this would be loaded via joblib/pickle after training on Postgres historical data
clf = RandomForestClassifier(random_state=42)
# X: [Avg Days to Pay, Missed Payments, Total Order Volume (UZS millions)]
# Y: 1 (Eligible for Increase), 0 (Not Eligible)
X_train = [[2.5, 0, 15.0], [14.0, 3, 2.0], [5.0, 0, 50.0], [30.0, 5, 1.0]]
Y_train = [1, 0, 1, 0]
clf.fit(X_train, Y_train)

@app.route('/credit/score', methods=['POST'])
def score_credit():
    data = request.json
    retailer_id = data.get('retailer_id')
    
    # Extract features (in real-world, we query Postgres/Spanner here based on retailer_id)
    avg_days_to_pay = data.get('avg_days_to_pay', 5.0)
    missed_payments = data.get('missed_payments', 0)
    total_volume_millions = data.get('total_volume_millions', 10.0)
    
    features = [[avg_days_to_pay, missed_payments, total_volume_millions]]
    prediction = clf.predict(features)[0]
    probability = clf.predict_proba(features)[0][1]
    
    suggested_increase = 0
    if prediction == 1:
        # Suggest a 20% increase on their base credit line if they are reliable
        current_limit = data.get('current_limit_uzs', 1000000)
        suggested_increase = int(current_limit * 0.20)

    return jsonify({
        "retailer_id": retailer_id,
        "eligible_for_increase": bool(prediction == 1),
        "confidence_score": round(float(probability), 3),
        "suggested_increase_uzs": suggested_increase
    })

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8081))
    app.run(host='0.0.0.0', port=port)
