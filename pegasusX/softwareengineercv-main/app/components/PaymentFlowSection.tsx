'use client';

import Link from 'next/link';
import ChamferButton, { ChamferArrowIcon } from './ChamferButton';
import PageSection from './layout/PageSection';
import PaymentFromToLogos from './payment/PaymentFromToLogos';
import { useLanguage } from '../context/LanguageContext';

export default function PaymentFlowSection() {
  const { t } = useLanguage();

  return (
    <PageSection aria-labelledby="payment-flow-heading" className="!pb-0 md:!pb-2">
      <div className="payment-flow-card payment-flow-card--half">
        <div className="payment-flow-card__body">
          <p className="editorial-eyebrow">
            {t('pay_flow_eyebrow', 'Finance · Payment integrity')}
          </p>
          <h2 id="payment-flow-heading" className="payment-flow-card__title">
            {t('pay_flow_title', 'How pay-at-delivery works')}
          </h2>
          <p className="payment-flow-card__description">
            {t(
              'pay_flow_desc',
              'Retailers checkout without a charge. Drivers collect cash or card when they arrive. Treasury reconciles every stop — duplicate protection and a clear audit trail for suppliers.'
            )}
          </p>
          <ChamferButton href="/capabilities/payment-confidence" variant="fill" className="mt-8">
            {t('pay_flow_cta', 'Payment confidence')}
          </ChamferButton>
        </div>

        <div className="payment-flow-card__visual" aria-hidden={false}>
          <PaymentFromToLogos
            variant="from-only"
            size="feature"
            fromPlace={t('pay_flow_from', 'Retailer checkout')}
          />
        </div>

        <Link href="/projects/payment-integrity" className="payment-flow-card__footer">
          <span className="chamfer-card__meta">
            {t('pay_flow_footer', 'Capabilities · Payment Confidence')}
          </span>
          <span className="chamfer-btn chamfer-btn--ghost chamfer-btn--icon" aria-hidden>
            <ChamferArrowIcon />
          </span>
        </Link>
      </div>
    </PageSection>
  );
}
