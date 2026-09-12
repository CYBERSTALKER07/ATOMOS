import React from 'react';
import { AlertTriangle } from 'lucide-react';

interface ShelfAlertModalProps {
  visible: boolean;
  productId: string;
  currentStock: number;
  onDismiss: () => void;
}

export const ShelfAlertModal: React.FC<ShelfAlertModalProps> = ({
  visible,
  productId,
  currentStock,
  onDismiss
}) => {
  if (!visible) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-red-50 p-6 rounded-lg shadow-xl max-w-sm w-full text-center border border-red-200">
        <AlertTriangle className="w-12 h-12 text-red-600 mx-auto mb-4" />
        <h2 className="text-xl font-bold text-red-800 mb-2">RESTOCK NEEDED</h2>
        <p className="text-red-900 mb-6">
          Product: <span className="font-mono bg-red-100 px-1 rounded">{productId}</span> has dropped to {currentStock} items on the floor.
        </p>
        <button 
          onClick={onDismiss}
          className="w-full bg-red-600 hover:bg-red-700 text-white font-bold py-2 px-4 rounded transition-colors"
        >
          ACKNOWLEDGE
        </button>
      </div>
    </div>
  );
};
