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
        className="flex items-center justify-between w-full min-h-11 px-4 rounded-md border text-left transition-colors focus:outline-none focus-visible:ring-2"
        style={{
          background: 'var(--desk-surface)',
          color: 'var(--desk-text-primary)',
          borderColor: 'var(--desk-border)',
          boxShadow: 'none',
        }}
      >
        <span className="flex flex-col">
          <span className="text-xs font-medium uppercase tracking-wider" style={{ color: 'var(--desk-text-tertiary)' }}>{label}</span>
          <span className="text-lg">
            {selected ? selected.toLocaleDateString() : 'Pick a date...'}
          </span>
        </span>
        <Icon name="calendar" className="w-6 h-6" />
      </button>

      {/* Popover */}
      {isOpen && (
        <div 
          className="absolute z-50 mt-2 p-4 w-85 border fade-in-up origin-top"
          style={{
            background: 'var(--desk-surface)',
            borderColor: 'var(--desk-border)',
            borderRadius: 'var(--radius-lg)',
            boxShadow: '0 12px 28px rgba(15, 23, 42, 0.16)',
          }}
        >
          {/* Header */}
          <div className="flex justify-between items-center mb-4">
            <button className="md-icon-btn rounded-full p-2 transition-colors" style={{ color: 'var(--desk-text-secondary)' }} type="button" onClick={handlePrevMonth}>
               <Icon name="left" className="w-5 h-5"/>
            </button>
            <div className="text-xl font-semibold" style={{ color: 'var(--desk-text-primary)' }}>
              {monthNames[currentMonth]} {currentYear}
            </div>
            <button className="md-icon-btn rounded-full p-2 transition-colors" style={{ color: 'var(--desk-text-secondary)' }} type="button" onClick={handleNextMonth}>
               <Icon name="right" className="w-5 h-5" />
            </button>
          </div>

          {/* Grid */}
          <div className="grid grid-cols-7 gap-2 text-center text-sm font-medium mb-2" style={{ color: 'var(--desk-text-tertiary)' }}>
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
                  className={`h-10 w-10 rounded-full flex items-center justify-center text-base font-semibold transition-colors ${!isSelected && !isToday ? 'hover:bg-(--desk-surface-subtle)' : ''}`}
                  style={isSelected
                    ? { background: 'var(--desk-accent)', color: 'var(--desk-surface)' }
                    : isToday
                      ? { background: 'var(--desk-accent-soft)', color: 'var(--desk-text-primary)' }
                      : { color: 'var(--desk-text-primary)' }}
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