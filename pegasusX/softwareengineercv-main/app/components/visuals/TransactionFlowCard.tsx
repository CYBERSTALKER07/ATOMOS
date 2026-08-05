'use client';

export default function TransactionFlowCard() {
  return (
    <div className="transaction-flow-card">
      <div className="transaction-flow-card__head">
        <div>
          <h3 className="transaction-flow-card__title">Transactions Review</h3>
          <p className="transaction-flow-card__sub">
            Access and manage fulfillment transactions across the last two fiscal years.
          </p>
        </div>
        <div className="transaction-flow-card__controls">
          <span className="transaction-flow-card__pill">Yearly</span>
        </div>
      </div>

      <div className="transaction-flow-card__stats">
        <div>
          <p className="transaction-flow-card__stat-label">Overall transaction</p>
          <p className="transaction-flow-card__stat-value">$456,245</p>
        </div>
        <div>
          <p className="transaction-flow-card__stat-label">Monthly</p>
          <p className="transaction-flow-card__stat-value">$36,962</p>
        </div>
      </div>

      <div className="transaction-flow-card__viz" aria-hidden>
        <div className="transaction-flow-card__sources">
          <div className="transaction-flow-card__source">
            <span>27%</span>
            <p>Deposits</p>
          </div>
          <div className="transaction-flow-card__source">
            <span>73%</span>
            <p>Subscription · Sent to · Monthly plan</p>
          </div>
        </div>
        <svg className="transaction-flow-card__paths" viewBox="0 0 400 200" preserveAspectRatio="none">
          <path d="M80 50 C180 50 220 80 320 40" stroke="rgba(255,255,255,0.2)" strokeWidth="8" fill="none" />
          <path d="M80 150 C200 120 240 100 320 90" stroke="rgba(255,255,255,0.15)" strokeWidth="14" fill="none" />
          <path d="M80 150 C180 160 220 170 320 160" stroke="rgba(255,255,255,0.12)" strokeWidth="10" fill="none" />
        </svg>
        <div className="transaction-flow-card__dests">
          <div className="transaction-flow-card__dest">
            <span>$79,245</span>
            <div className="transaction-flow-card__avatar" />
          </div>
          <div className="transaction-flow-card__dest">
            <span>$120.65</span>
            <div className="transaction-flow-card__brands">◎ ◎ ◎</div>
          </div>
          <div className="transaction-flow-card__dest">
            <span>$134.78</span>
            <div className="transaction-flow-card__pill-icon">U</div>
          </div>
        </div>
      </div>
    </div>
  );
}
