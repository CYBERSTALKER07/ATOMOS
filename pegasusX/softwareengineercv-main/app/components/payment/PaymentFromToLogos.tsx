import BusinessBagIcon from '../icons/BusinessBagIcon';
import DebitCardIcon from '../icons/DebitCardIcon';
import MapPinIcon from '../icons/MapPinIcon';

type PaymentFromToLogosProps = {
  fromLabel?: string;
  toLabel?: string;
  fromPlace?: string;
  toPlace?: string;
  size?: 'card' | 'feature';
  variant?: 'from-to' | 'from-only';
  toIcon?: 'card' | 'pin';
  className?: string;
};

export default function PaymentFromToLogos({
  fromLabel = 'From',
  toLabel = 'To',
  fromPlace = 'Retailer checkout',
  toPlace = 'Delivery stop',
  size = 'card',
  variant = 'from-to',
  toIcon = 'card',
  className = '',
}: PaymentFromToLogosProps) {
  const iconSize = size === 'feature' ? 128 : 88;

  if (variant === 'from-only') {
    return (
      <div
        className={`payment-from-to-logos payment-from-to-logos--from-only payment-from-to-logos--${size} ${className}`.trim()}
        aria-label={`From ${fromPlace}`}
      >
        <div className="payment-from-to-logos__mark payment-from-to-logos__mark--hero" aria-hidden>
          <BusinessBagIcon
            size={iconSize}
            className="payment-from-to-logos__icon payment-from-to-logos__icon--bag"
          />
        </div>
        <div className="payment-from-to-logos__copy payment-from-to-logos__copy--centered">
          <span className="payment-from-to-logos__label">{fromLabel}</span>
          <span className="payment-from-to-logos__place">{fromPlace}</span>
        </div>
      </div>
    );
  }

  const ToIcon = toIcon === 'pin' ? MapPinIcon : DebitCardIcon;

  return (
    <div
      className={`payment-from-to-logos payment-from-to-logos--${size} ${className}`.trim()}
      aria-label={`Payment flow from ${fromPlace} to ${toPlace}`}
    >
      <div className="payment-from-to-logos__end payment-from-to-logos__end--from">
        <div className="payment-from-to-logos__mark" aria-hidden>
          <BusinessBagIcon
            size={iconSize}
            className="payment-from-to-logos__icon payment-from-to-logos__icon--bag"
          />
        </div>
        <div className="payment-from-to-logos__copy">
          <span className="payment-from-to-logos__label">{fromLabel}</span>
          <span className="payment-from-to-logos__place">{fromPlace}</span>
        </div>
      </div>

      <div className="payment-from-to-logos__bridge" aria-hidden>
        <span className="payment-from-to-logos__track" />
        <span className="payment-from-to-logos__arrow">→</span>
      </div>

      <div className="payment-from-to-logos__end payment-from-to-logos__end--to">
        <div className="payment-from-to-logos__mark" aria-hidden>
          <ToIcon
            size={iconSize}
            className={`payment-from-to-logos__icon payment-from-to-logos__icon--${toIcon === 'pin' ? 'pin' : 'card'}`}
          />
        </div>
        <div className="payment-from-to-logos__copy">
          <span className="payment-from-to-logos__label">{toLabel}</span>
          <span className="payment-from-to-logos__place">{toPlace}</span>
        </div>
      </div>
    </div>
  );
}
