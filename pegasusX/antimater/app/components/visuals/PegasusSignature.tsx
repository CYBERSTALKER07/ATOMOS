import React from 'react';

export default function PegasusSignature({ className = "" }: { className?: string }) {
  return (
    <svg 
      viewBox="0 0 400 150" 
      className={className}
      fill="none" 
      stroke="currentColor" 
      strokeWidth="2.5" 
      strokeLinecap="round" 
      strokeLinejoin="round"
    >
      {/* 
        Signature "Pegasus" modeled after Oprah Winfrey's signature.
        Features a massive swooping P that circles back, cursive internal letters, 
        and a dramatic descending tail on the final 's'
      */}
      <path d="
        M 120,80 
        C 120,40 100,20 60,20 
        C 20,20 10,60 10,90
        C 10,130 50,140 90,140
        C 160,140 200,90 200,50
        C 200,30 180,10 150,10
        C 130,10 110,30 110,60
        C 110,80 120,90 140,90
        C 160,90 170,70 170,70

        M 170,70
        C 170,70 175,60 185,60
        C 190,60 195,65 195,75
        C 195,85 180,95 185,95
        C 195,95 205,70 205,70

        M 205,70
        C 205,70 200,85 200,95
        C 200,120 180,130 180,110
        C 180,95 210,65 210,65

        M 210,65
        C 210,65 205,80 215,80
        C 225,80 225,65 225,65

        M 225,65
        C 225,65 220,90 230,90
        C 240,90 240,75 240,75

        M 240,75
        C 240,75 235,90 250,90
        C 265,90 265,70 265,70

        M 265,70
        C 265,70 260,95 270,95
        C 285,95 300,110 300,130
        C 300,160 270,160 270,140
        C 270,120 300,90 320,90
        C 340,90 350,120 380,120
      " />
    </svg>
  );
}
