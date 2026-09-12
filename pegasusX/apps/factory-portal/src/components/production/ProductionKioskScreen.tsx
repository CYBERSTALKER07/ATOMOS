import React, { useState } from 'react';

export const ProductionKioskScreen: React.FC = () => {
  const machineId = "MAC-123";
  const [isJammed, setIsJammed] = useState(false);
  const [currentStatus, setCurrentStatus] = useState("IDLE");

  const bgColor = isJammed ? "bg-red-600" : "bg-gray-50";
  const textColor = isJammed ? "text-white" : "text-gray-900";

  return (
    <div className={`min-h-screen flex flex-col items-center justify-center p-8 transition-colors duration-300 ${bgColor}`}>
      <h1 className={`text-3xl mb-8 ${textColor}`}>Production Kiosk - {machineId}</h1>
      
      <div className={`text-6xl font-bold mb-16 ${textColor}`}>
        STATUS: {currentStatus}
      </div>

      <div className="flex space-x-8">
        <button
          onClick={() => {
            setCurrentStatus("IN_PRODUCTION");
            setIsJammed(false);
          }}
          className="w-64 h-40 bg-green-500 hover:bg-green-600 text-white text-3xl font-bold rounded-2xl shadow-lg transition-transform active:scale-95"
        >
          START RUN
        </button>

        <button
          onClick={() => {
            setCurrentStatus("PAUSED");
            setIsJammed(false);
          }}
          className="w-64 h-40 bg-yellow-500 hover:bg-yellow-600 text-white text-3xl font-bold rounded-2xl shadow-lg transition-transform active:scale-95"
        >
          PAUSE
        </button>

        <button
          onClick={() => {
            setCurrentStatus("JAMMED");
            setIsJammed(true);
          }}
          className="w-64 h-40 bg-red-700 hover:bg-red-800 text-white text-3xl font-bold rounded-2xl shadow-lg transition-transform active:scale-95"
        >
          FLAG ISSUE
        </button>
      </div>
    </div>
  );
};
