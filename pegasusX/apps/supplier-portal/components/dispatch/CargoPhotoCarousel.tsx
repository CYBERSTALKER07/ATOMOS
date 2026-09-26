"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";
import { PoDPhotoReport } from "@pegasusx/types";

interface CargoPhotoCarouselProps {
  photos?: PoDPhotoReport[];
  onRequestPhoto?: () => void;
}

export const CargoPhotoCarousel: React.FC<CargoPhotoCarouselProps> = ({
  photos = [],
  onRequestPhoto,
}) => {
  const t = usePortalT();
  return (
    <div className="bg-[#121417] border border-gray-800 rounded-2xl p-5 select-none shadow-lg">
      <h3 className="text-sm font-semibold text-gray-300 tracking-wide uppercase mb-4">{t("supplier_portal.dispatch.cargo_photo_carousel.text.cargo_photo_reports")}</h3>

      <div className="flex items-center gap-4 overflow-x-auto pb-2 custom-scrollbar">
        {photos.length === 0 ? (
          <p className="text-xs text-gray-500 py-8 px-2">
            No cargo photos for this route yet.
          </p>
        ) : (
          photos.map((item) => (
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
          ))
        )}

        <button
          onClick={onRequestPhoto}
          className="shrink-0 w-32 h-40 border-2 border-dashed border-gray-800 hover:border-blue-500 rounded-xl flex flex-col items-center justify-center text-xs text-gray-400 hover:text-blue-400 transition-all bg-gray-900/30 hover:bg-blue-950/20"
        >
          <div className="w-8 h-8 rounded-full bg-gray-800 text-gray-300 flex items-center justify-center mb-1 text-sm">
            +
          </div>
          <span className="font-semibold text-[11px]">{t("supplier_portal.dispatch.cargo_photo_carousel.text.request_photo")}</span>
        </button>
      </div>
    </div>
  );
};
