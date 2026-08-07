'use client';

import { useLanguage } from '../context/LanguageContext';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Image from 'next/image';
import Link from 'next/link';

interface MasonryGridProps {
  images: {
    src: string;
    title: string;
    category: string;
    slug: string;
    color: string;
  }[];
}

export default function MasonryGrid({ images }: MasonryGridProps) {
  const { t } = useLanguage();

  const gridRef = useRef<HTMLDivElement>(null);
  const [imagesLoaded, setImagesLoaded] = useState(false);

  useEffect(() => {
    // Preload images
    let loadedCount = 0;
    const totalImages = images.length;

    images.forEach((img) => {
      const image = new window.Image();
      image.src = img.src;
      image.onload = () => {
        loadedCount++;
        if (loadedCount === totalImages) {
          setImagesLoaded(true);
        }
      };
      image.onerror = () => {
        loadedCount++;
        if (loadedCount === totalImages) {
          setImagesLoaded(true);
        }
      };
    });
  }, [images]);

  useEffect(() => {
    if (!imagesLoaded || !gridRef.current) return;

    const items = gridRef.current.querySelectorAll('.masonry-item');
    
    gsap.fromTo(
      items,
      {
        opacity: 0,
        y: 60,
        scale: 0.8,
      },
      {
        opacity: 1,
        y: 0,
        scale: 1,
        duration: 0.8,
        stagger: 0.08,
        ease: 'power3.out',
      }
    );
  }, [imagesLoaded]);

  const getColorClass = (color: string) => {
    const colorMap: { [key: string]: string } = {
      '#FFA500': 'hover-orange',
      '#FBFF63': 'hover-yellow',
      '#A9EBF9': 'hover-cyan',
      '#8DDC96': 'hover-green',
      '#BDE7FF': 'hover-blue',
      '#DABDFF': 'hover-purple',
      '#FFDA6F': 'hover-gold',
      '#FE5934': 'hover-red'
    };
    return colorMap[color] || 'hover-cyan';
  };

  return (
    <div
      ref={gridRef}
      className="columns-1 sm:columns-2 lg:columns-3 xl:columns-4 gap-6 md:gap-8 space-y-6 md:space-y-8"
    >
      {images.map((image, index) => (
        <Link
          key={index}
          href={`/projects/${image.slug}`}
          className={`masonry-item group block break-inside-avoid mb-6 md:mb-8 relative overflow-hidden rounded-3xl border-2 border-white bg-black transition-all duration-300 focus-visible:outline-none ${getColorClass(
            image.color
          )}`}
        >
          <div className="relative overflow-hidden">
            {/* Image */}
            <div className="relative w-full aspect-auto">
              <Image
                src={image.src}
                alt={image.title}
                width={600}
                height={800}
                className="w-full h-auto object-cover transition-transform duration-500 group-hover:scale-105"
              />
              
              {/* Overlay on hover */}
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/60 transition-all duration-300 flex items-end p-6">
                <div className="transform translate-y-full group-hover:translate-y-0 transition-transform duration-300">
                  <span className="inline-block px-3 py-1 mb-3 bg-white/20 backdrop-blur-sm border border-white/30 rounded-full text-xs font-light text-white">
                    {image.category}
                  </span>
                  <h3 className="text-xl md:text-2xl font-light text-white mb-2">
                    {image.title}
                  </h3>
                  <div className="flex items-center gap-2 text-white">
                    <span className="text-sm">{t('btn_view_project', 'View Project')}</span>
                    <span className="text-lg">→</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </Link>
      ))}
    </div>
  );
}
