'use client';

import React, { useState } from 'react';
import { useClickOutside } from '../../hooks/useClickOutside';
import Icon from '../Icon';

export default function M3DatePicker({
  selected,
  onChange,
  label = 'Select date',
}: {
  selected?: Date | null;
  onChange: (d: Date) => void;
  label?: string;
}) {
  const [isOpen, setIsOpen] = useState(false);
  
  const [viewDate, setViewDate] = useState(selected || new Date());
  
  const ref = useClickOutside<HTMLDivElement>(() => setIsOpen(false));

  const currentYear = viewDate.getFullYear();
  const currentMonth = viewDate.getMonth();

  const daysInMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
  const startDayPadding = new Date(currentYear, currentMonth, 1).getDay();
  
  const handlePrevMonth = () => setViewDate(new Date(currentYear, currentMonth - 1, 1));
  const handleNextMonth = () => setViewDate(new Date(currentYear, currentMonth + 1, 1));

  const handleSelectDate = (day: number) => {
    onChange(new Date(currentYear, currentMonth, day));
    setIsOpen(false);
  };

  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ];

  return (
    <div className="relative w-full" ref={ref}>
      {/* Tap Target (Bigger for Touch / M3 UX Guidelines) */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="
          flex items-center justify-between w-full min-h-[56px] px-4 
          bg-[var(--surface)] rounded-2xl
          text-[var(--foreground)] text-left
          border border-[var(--border)]
          focus:outline-none focus:ring-2 focus:ring-[var(--accent)]
          transition-all
        "
      >
        <span className="flex flex-col">
          <span className="text-xs text-[var(--accent)] font-medium uppercase tracking-wider">{label}</span>
          <span className="text-lg">
            {selected ? selected.toLocaleDateString() : 'Pick a date...'}
          </span>
        </span>
        <Icon name="calendar" className="text-[var(--accent)] w-6 h-6" />
      </button>

      {/* Popover */}
      {isOpen && (
        <div 
          className="
            absolute z-50 mt-2 p-6 w-[340px] 
            bg-[var(--surface)] 
            rounded-3xl shadow-xl md-elevation-4
            border border-[var(--border)]
            fade-in-up origin-top
          "
        >
          {/* Header */}
          <div className="flex justify-between items-center mb-6">
            <button className="md-icon-btn bg-[var(--surface)] rounded-full hover:bg-[var(--surface)] p-2 transition-colors" type="button" onClick={handlePrevMonth}>
               <Icon name="left" className="w-5 h-5"/>
            </button>
            <div className="text-xl font-bold text-[var(--foreground)]">
              {monthNames[currentMonth]} {currentYear}
            </div>
            <button className="md-icon-btn bg-[var(--surface)] rounded-full hover:bg-[var(--surface)] p-2 transition-colors" type="button" onClick={handleNextMonth}>
               <Icon name="right" className="w-5 h-5" />
            </button>
          </div>

          {/* Grid */}
          <div className="grid grid-cols-7 gap-2 text-center text-sm font-medium text-[var(--muted)] mb-2">
            <div>S</div><div>M</div><div>T</div><div>W</div><div>T</div><div>F</div><div>S</div>
          </div>
          <div className="grid grid-cols-7 gap-2">
            {/* Blank padding cells */}
            {Array.from({ length: startDayPadding }).map((_, i) => (
              <div key={`pad-${i}`} className="h-10 w-10" />
            ))}

            {/* Days cells */}
            {Array.from({ length: daysInMonth }).map((_, i) => {
              const dayNum = i + 1;
              const isSelected = selected?.getDate() === dayNum && selected?.getMonth() === currentMonth;
              const isToday = new Date().getDate() === dayNum && new Date().getMonth() === currentMonth;
              
              return (
                <button
                  key={dayNum}
                  type="button"
                  onClick={() => handleSelectDate(dayNum)}
                  className={`
                    h-10 w-10 rounded-full flex items-center justify-center
                    text-base font-semibold transition-all
                    ${isSelected 
                      ? 'bg-[var(--accent)] text-[var(--accent-foreground)] shadow-md transform scale-110' 
                      : isToday 
                        ? 'bg-[var(--accent-soft)] text-[var(--accent-soft-foreground)]'
                        : 'text-[var(--foreground)] hover:bg-[var(--surface)]'
                    }
                  `}
                >
                  {dayNum}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}