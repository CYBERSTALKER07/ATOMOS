'use client';

import { X, ShoppingBag, CreditCard, ChevronRight } from 'lucide-react';
import { useCart } from '../lib/cart';

interface CartDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  onCheckout: () => void;
}

export default function CartDrawer({ isOpen, onClose, onCheckout }: CartDrawerProps) {
  const { items, updateQuantity, removeFromCart, total } = useCart();

  if (isOpen === false) return null;

  return (
    <>
      <div 
        className="fixed inset-0 bg-black/40 backdrop-blur-sm z-40 transition-opacity"
        onClick={onClose}
      />

      <div className="fixed top-0 right-0 w-[440px] h-full shadow-2xl z-50 flex flex-col transition-transform transform translate-x-0 border-l border-[var(--border)]" style={{ background: 'var(--background)' }}>
        
        <div className="p-6 flex justify-between items-center border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
          <div className="flex items-center gap-3" style={{ color: 'var(--accent)' }}>
            <ShoppingBag size={24} />
            <h2 className="md-typescale-title-large font-bold text-foreground">Your Cart</h2>
          </div>
          <button onClick={onClose} className="p-2 rounded-full hover:bg-surface text-muted">
            <X size={20} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 flex flex-col gap-6">
          {items.map((item) => (
            <div key={item.product_id} className="flex gap-4 border-b border-[var(--border)] pb-6">
              <div className="w-20 h-20 rounded-xl flex-shrink-0" style={{ background: 'var(--surface)' }} />
              <div className="flex-1 flex flex-col justify-between">
                <h4 className="md-typescale-title-small font-bold text-foreground leading-snug line-clamp-2">
                  {item.name}
                </h4>
                
                <div className="flex items-center justify-between mt-3">
                  <div className="flex items-center gap-3 rounded-full px-2 py-1" style={{ background: 'var(--surface)' }}>
                    <button 
                      onClick={() => updateQuantity(item.product_id, item.quantity - 1)}
                      className="w-7 h-7 rounded-full shadow-sm font-medium text-foreground" style={{ background: 'var(--background)' }}
                    >
                      -
                    </button>
                    <span className="md-typescale-body-medium font-semibold w-4 text-center">{item.quantity}</span>
                    <button 
                      onClick={() => updateQuantity(item.product_id, item.quantity + 1)}
                      className="w-7 h-7 rounded-full shadow-sm font-medium"
                      style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
                    >
                      +
                    </button>
                  </div>
                  <span className="md-typescale-title-medium font-extrabold" style={{ color: 'var(--accent)' }}>
                    {(item.price * item.quantity).toLocaleString()}
                  </span>
                </div>
              </div>
            </div>
          ))}
          {items.length === 0 && (
            <div className="text-center text-muted mt-10">Cart is empty</div>
          )}
        </div>

        <div className="p-6 border-t border-[var(--border)]" style={{ background: 'var(--surface)' }}>
          <div className="flex justify-between mb-2 md-typescale-body-large text-muted">
             <span>Subtotal</span>
             <span>{total.toLocaleString()}</span>
          </div>
          <div className="flex justify-between mb-6 md-typescale-headline-small font-bold text-foreground">
             <span>Total</span>
             <span>{total.toLocaleString()}</span>
          </div>
          
          <button 
            disabled={items.length === 0}
            onClick={() => onCheckout()}
            className="w-full py-4 rounded-full flex items-center justify-center gap-2 md-typescale-label-large font-bold shadow-md transition-transform hover:-translate-y-0.5 disabled:opacity-50 disabled:hover:translate-y-0"
            style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-foreground)' }}
          >
            <CreditCard size={20} />
            Checkout {total.toLocaleString()}
            <ChevronRight size={20} />
          </button>
        </div>

      </div>
    </>
  );
}
