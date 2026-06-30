# SEO Configuration Guide

## Overview
This document outlines the comprehensive SEO setup for your Next.js portfolio application.

## What Has Been Configured

### 1. **Metadata Management**
- ✅ Global metadata in `app/layout.tsx`
- ✅ Page-specific metadata for all routes (home, projects, resume, join)
- ✅ Open Graph tags for social media sharing
- ✅ Twitter Card support
- ✅ Meta keywords and descriptions

### 2. **Technical SEO**
- ✅ **Sitemap** (`app/sitemap.ts`) - Automatically generated at `/sitemap.xml`
- ✅ **Robots.txt** (`app/robots.ts`) - Automatically generated at `/robots.txt`
- ✅ **Structured Data (JSON-LD)** - Person and Website schemas on homepage
- ✅ **Web Manifest** (`public/site.webmanifest`) - PWA support

### 3. **Performance Optimizations**
- ✅ Next.js Image component (already in use via Next.js)
- ✅ Automatic code splitting
- ✅ Mobile responsiveness
- ✅ GSAP animations for better UX

### 4. **Accessibility & SEO Best Practices**
- ✅ Semantic HTML structure
- ✅ Alt text for images
- ✅ ARIA labels on buttons
- ✅ Proper heading hierarchy
- ✅ Canonical URLs

## Configuration Steps

### Step 1: Set Your Site URL
Create a `.env.local` file in the root directory:

```bash
NEXT_PUBLIC_SITE_URL=https://softwareengineercv.vercel.app
```

Replace `https://yourportfolio.com` with your actual domain.

### Step 2: Update Personal Information
Update the following files with your information:

**`app/layout.tsx`:**
- Replace `'Your Name'` in authors, creator, publisher
- Replace `'@yourtwitterhandle'` with your Twitter handle
- Replace `'your-google-verification-code'` with your actual Google Search Console verification code

**`app/page.tsx`:**
- Update the Person schema:
  - `name: 'Your Name'`
  - `sameAs` array with your social media links
  - `alumniOf` with your university name

### Step 3: Add Open Graph Image
Create an Open Graph image (`og-image.png`) at 1200x630px and place it in the `public` folder. This image will be shown when your site is shared on social media.

### Step 4: Google Search Console Setup
1. Go to [Google Search Console](https://search.google.com/search-console)
2. Add your property (domain or URL prefix)
3. Get your verification code
4. Add it to `app/layout.tsx` in the verification section

### Step 5: Submit Your Sitemap
After deployment:
1. Your sitemap will be available at: `https://yoursite.com/sitemap.xml`
2. Submit it to Google Search Console
3. Submit to Bing Webmaster Tools (optional)

## SEO Checklist

- [ ] Set `NEXT_PUBLIC_SITE_URL` in `.env.local`
- [ ] Update all personal information in metadata
- [ ] Create and add `og-image.png` (1200x630px)
- [ ] Update social media links in structured data
- [ ] Verify site with Google Search Console
- [ ] Submit sitemap to search engines
- [ ] Test Open Graph tags with [OpenGraph.xyz](https://www.opengraph.xyz/)
- [ ] Test Twitter Cards with [Twitter Card Validator](https://cards-dev.twitter.com/validator)
- [ ] Run Lighthouse audit in Chrome DevTools
- [ ] Test mobile responsiveness
- [ ] Verify structured data with [Rich Results Test](https://search.google.com/test/rich-results)

## Next.js SEO Features Used

1. **App Router Metadata API** - Type-safe metadata configuration
2. **Dynamic Sitemap Generation** - Automatically updates with routes
3. **Robots.txt Generation** - Controls search engine crawling
4. **next-seo Package** - Installed for future advanced SEO needs

## Testing Your SEO

Run these commands to test your site:

```bash
# Build for production
npm run build

# Start production server
npm start
```

Visit these URLs to verify:
- `http://localhost:3000/sitemap.xml`
- `http://localhost:3000/robots.txt`
- `http://localhost:3000/site.webmanifest`

## Performance Tips

1. **Image Optimization**: Use Next.js Image component for all images
2. **Font Loading**: Already using next/font for optimized font loading
3. **Code Splitting**: Automatic with Next.js App Router
4. **Caching**: Configure proper cache headers in production
5. **CDN**: Deploy to Vercel or Netlify for automatic CDN

## Monitoring

After deployment, monitor:
- Google Search Console for indexing status
- Google Analytics for traffic
- Core Web Vitals in Chrome DevTools
- PageSpeed Insights scores

## Resources

- [Next.js SEO Documentation](https://nextjs.org/docs/app/building-your-application/optimizing/metadata)
- [Google Search Central](https://developers.google.com/search)
- [Schema.org](https://schema.org/)
- [Open Graph Protocol](https://ogp.me/)

---

**Last Updated:** October 12, 2025
