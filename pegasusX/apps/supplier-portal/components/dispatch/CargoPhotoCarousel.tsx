"use client";

import React from "react";
import { PoDPhotoReport } from "@pegasusx/types";

interface CargoPhotoCarouselProps {
  photos?: PoDPhotoReport[];
  onRequestPhoto?: () => void;
}

export const CargoPhotoCarousel: React.FC<CargoPhotoCarouselProps> = ({
  photos = [
    {
      id: "p1",
      title: "Point #1 Cargo Photo",
      location_name: "0578 Mraz Lock",
      timestamp: "08:03 AM",
      photo_url: "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=300&q=80",
      step_number: 1,
    },
    {
      id: "p2",
      title: "Point #1 Cargo Photo",
      location_name: "4164 Torrance Plaza",
      timestamp: "08:33 AM",
      photo_url: "https://images.unsplash.com/photo-1578575437130-527eed3abbec?auto=format&fit=crop&w=300&q=80",
      step_number: 1,
    },
    {
      id: "p3",
      title: "Point #2 Cargo Photo",
      location_name: "0732 Allen Crossing",
      timestamp: "09:01 AM",
      photo_url: "https://images.unsplash.com/photo-1553413077-190dd305871c?auto=format&fit=crop&w=300&q=80",
      step_number: 2,
    },
    {
      id: "p4",
      title: "Point #3 Cargo Photo",
      location_name: "399 Lorine Island",
      timestamp: "09:21 AM",
      photo_url: "https://images.unsplash.com/photo-1616401784845-180882ba9ba8?auto=format&fit=crop&w=300&q=80",
      step_number: 3,
    },
  ],
  onRequestPhoto,
}) => {
  return (
    <div className="bg-[#121417] border border-gray-800 rounded-2xl p-5 select-none shadow-lg">
      <h3 className="text-sm font-semibold text-gray-300 tracking-wide uppercase mb-4">Cargo Photo Reports</h3>

      <div className="flex items-center gap-4 overflow-x-auto pb-2 custom-scrollbar">
        {photos.map((item) => (
          <div key={item.id} className="shrink-0 w-44 bg-[#181b20] border border-gray-800 rounded-xl p-2 group hover:border-gray-700 transition-colors">
            <div className="h-28 w-full rounded-lg overflow-hidden relative bg-gray-900 mb-2">
              <img
                src={item.photo_url}
                alt={item.title}
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
              />
              <span className="absolute bottom-1 right-1 bg-black/70 text-white text-[9px] font-semibold px-1.5 py-0.5 rounded">
                {item.timestamp}
              </span>
            </div>
            <p className="text-[11px] font-bold text-white truncate">{item.title}</p>
            <p className="text-[10px] text-gray-400 truncate">{item.location_name}</p>
          </div>
        ))}

        {/* Request Photo Action Button Box */}
        <button
          onClick={onRequestPhoto}
          className="shrink-0 w-32 h-40 border-2 border-dashed border-gray-800 hover:border-blue-500 rounded-xl flex flex-col items-center justify-center text-xs text-gray-400 hover:text-blue-400 transition-all bg-gray-900/30 hover:bg-blue-950/20"
        >
          <div className="w-8 h-8 rounded-full bg-gray-800 text-gray-300 flex items-center justify-center mb-1 text-sm">
            📷
          </div>
          <span className="font-semibold text-[11px]">Request Photo</span>
        </button>
      </div>
    </div>
  );
};
