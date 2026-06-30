'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { projects } from '../data/projects';
import PillNav from '../components/PillNav';
import Link from 'next/link';

// Note: Metadata export won't work in client components
const pageMetadata = {
  title: 'Resume | Shakhzod Soliyev - Software Engineer',
  description: 'Professional resume of Shakhzod Soliyev - Software Engineer with expertise in React, Next.js, and full-stack web development.',
};

export default function ResumePage() {
  const resumeRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Set document title
    document.title = pageMetadata.title;
    
    // Add print styles
    const style = document.createElement('style');
    style.textContent = `
      @media print {
        body {
          background: white !important;
          color: black !important;
        }
        .no-print {
          display: none !important;
        }
        .print-section {
          page-break-inside: avoid;
          break-inside: avoid;
        }
        .resume-container {
          max-width: 100% !important;
          padding: 40px !important;
          box-shadow: none !important;
        }
        a {
          color: black !important;
          text-decoration: none !important;
        }
        .border-white {
          border-color: black !important;
        }
        .text-white {
          color: black !important;
        }
        .bg-black {
          background: white !important;
        }
      }
    `;
    document.head.appendChild(style);
    
    if (headerRef.current) {
      gsap.fromTo(
        headerRef.current,
        { opacity: 0, y: -30 },
        { opacity: 1, y: 0, duration: 0.8, ease: 'power3.out' }
      );
    }
    
    return () => {
      document.head.removeChild(style);
    };
  }, []);

  const handleDownloadPDF = () => {
    if (typeof window !== 'undefined') {
      window.print();
    }
  };

  return (
    <>
      <div className="min-h-screen bg-black text-white no-print">
        <PillNav
          logo=""
          logoAlt="Portfolio Logo"
          items={[
            { label: 'Home', href: '/' },
            { label: 'About', href: '/#about' },
            { label: 'Skills', href: '/#skills' },
            { label: 'Projects', href: '/projects' },
            { label: 'Mobile Apps', href: '/mobile-apps' },
            { label: 'Web Apps', href: '/web-apps' },
            { label: 'Desktop Apps', href: '/desktop-apps' },
            { label: 'Resume', href: '/resume' },
            { label: 'Contact', href: '/#contact' }
          ]}
          activeHref="/resume"
          baseColor="#000000"
          pillColor="#ffffff"
          hoveredPillTextColor="#ffffff"
          pillTextColor="#000000"
        />

        {/* Action Buttons */}
        <div className="fixed top-24 right-8 z-50 flex flex-col gap-4 no-print">
          <button
            onClick={handleDownloadPDF}
            className="px-6 py-3 bg-white text-black border-2 border-white hover:bg-[#FFA500] hover:border-[#FFA500] transition-all duration-300 font-bold rounded-2xl shadow-lg"
          >
            📄 Download PDF
          </button>
          <Link
            href="/"
            className="px-6 py-3 bg-black text-white border-2 border-white hover:bg-white hover:text-black transition-all duration-300 font-bold rounded-2xl text-center"
          >
            ← Back Home
          </Link>
        </div>

        {/* Header */}
        <div ref={headerRef} className="text-center pt-32 pb-12 px-4 no-print">
          <h1 className="text-4xl md:text-6xl font-bold mb-4">Resume</h1>
          <div className="w-20 h-1 bg-white rounded-full mx-auto mb-6" />
          <p className="text-lg text-gray-300 max-w-2xl mx-auto">
            Professional Resume - Click &quot;Download PDF&quot; to save a copy
          </p>
        </div>
      </div>

      {/* Resume Content */}
      <div className="min-h-screen bg-white text-black py-12 px-4">
        <div
          ref={resumeRef}
          className="resume-container max-w-4xl mx-auto bg-white p-8 md:p-16 shadow-2xl rounded-2xl"
        >
          {/* Header Section */}
          <div className="text-center mb-12 print-section border-b-2 border-black pb-8">
            <h1 className="text-5xl md:text-7xl font-bold mb-3 tracking-tight">SHAKZHOD SOLIYEV</h1>
            <p className="text-xl md:text-2xl text-gray-700 mb-6 font-semibold">SOFTWARE ENGINEER</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm max-w-3xl mx-auto">
              <div className="flex items-center justify-center gap-2">
                <span className="font-bold">📧 Email:</span>
                <a href="mailto:shakhzodsoliyevmit@gmail.com" className="hover:text-[#FFA500] transition-colors">
                  shakhzodsoliyevmit@gmail.com
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-bold">📱 Phone:</span>
                <a href="tel:+990906047174" className="hover:text-[#FFA500] transition-colors">
                  +990 90 604 7174
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-bold">🌐 Portfolio:</span>
                <a href="https://66f552cfb4bb41102734c7f5--enchanting-custard-403d51.netlify.app" target="_blank" rel="noopener noreferrer" className="hover:text-[#FFA500] transition-colors truncate max-w-[250px]">
                  enchanting-custard-403d51.netlify.app
                </a>
              </div>
              <div className="flex items-center justify-center gap-2">
                <span className="font-bold">📍 Location:</span>
                <span>Tashkent City, Chilanzar 17</span>
              </div>
            </div>
          </div>

          {/* About Me Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-4 pb-2 border-b-2 border-black">
              ABOUT ME
            </h2>
            <p className="text-base leading-relaxed text-gray-800">
              I am a goal-driven frontend developer with a strong passion for creating user-friendly and visually appealing web applications. 
              I have experience with modern tools and technologies like React.js, Next.js, and JavaScript, and I am committed to delivering 
              high-quality, user-centric projects.
            </p>
          </div>

          {/* Education Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-6 pb-2 border-b-2 border-black">
              EDUCATION
            </h2>
            <div className="border-l-4 border-black pl-6">
              <div className="mb-2">
                <h3 className="text-xl font-bold">Bachelor of Software Engineering</h3>
                <p className="text-lg text-gray-700 font-semibold">American University of Technology</p>
              </div>
              <p className="text-gray-600 font-medium mb-2">2024 - 2028</p>
              <p className="text-base text-gray-800 leading-relaxed">
                Currently pursuing a degree in Software Engineering with focus on modern web development, 
                algorithms, and software architecture.
              </p>
            </div>
          </div>

          {/* Skills Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-6 pb-2 border-b-2 border-black">
              SKILLS
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <h3 className="text-lg font-bold mb-3 text-gray-800">Web Development</h3>
                <div className="flex flex-wrap gap-2">
                  {['React.js', 'Next.js', 'HTML', 'CSS', 'JavaScript'].map((skill, idx) => (
                    <span key={idx} className="px-4 py-2 bg-black text-white text-sm font-semibold rounded-lg hover:bg-[#FFA500] transition-colors cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-bold mb-3 text-gray-800">Design</h3>
                <div className="flex flex-wrap gap-2">
                  {['Figma', 'UI/UX Design', 'Responsive Design'].map((skill, idx) => (
                    <span key={idx} className="px-4 py-2 bg-black text-white text-sm font-semibold rounded-lg hover:bg-[#8DDC96] transition-colors cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-bold mb-3 text-gray-800">Digital Marketing</h3>
                <div className="flex flex-wrap gap-2">
                  {['SEO', 'Social Media Management'].map((skill, idx) => (
                    <span key={idx} className="px-4 py-2 bg-black text-white text-sm font-semibold rounded-lg hover:bg-[#A9EBF9] transition-colors cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-bold mb-3 text-gray-800">Mobile Development</h3>
                <div className="flex flex-wrap gap-2">
                  {['React Native'].map((skill, idx) => (
                    <span key={idx} className="px-4 py-2 bg-black text-white text-sm font-semibold rounded-lg hover:bg-[#DABDFF] transition-colors cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <h3 className="text-lg font-bold mb-3 text-gray-800">Backend & Tools</h3>
                <div className="flex flex-wrap gap-2">
                  {['Node.js', 'Git', 'GitHub'].map((skill, idx) => (
                    <span key={idx} className="px-4 py-2 bg-black text-white text-sm font-semibold rounded-lg hover:bg-[#FFDA6F] transition-colors cursor-default">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* Languages Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-6 pb-2 border-b-2 border-black">
              LANGUAGES
            </h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {[
                { lang: 'English', level: 'Advanced' },
                { lang: 'Russian', level: 'Fluent' },
                { lang: 'Uzbek', level: 'Native' },
                { lang: 'Chinese', level: 'Basic' }
              ].map((item, idx) => (
                <div key={idx} className="text-center p-4 border-2 border-black rounded-xl hover:bg-black hover:text-white transition-all duration-300">
                  <p className="font-bold text-lg">{item.lang}</p>
                  <p className="text-sm opacity-80">{item.level}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Work Experience Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-6 pb-2 border-b-2 border-black">
              WORK EXPERIENCE
            </h2>
            <div className="border-l-4 border-black pl-6 space-y-8">
              <div>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <h3 className="text-xl font-bold">Frontend Intern</h3>
                    <p className="text-gray-700 font-semibold">Marketing Company</p>
                  </div>
                  <span className="text-gray-600 font-medium whitespace-nowrap ml-4">2024 - 2025</span>
                </div>
                <ul className="list-disc list-inside space-y-2 text-gray-800">
                  <li>Participated in developing and maintaining web applications using HTML, CSS, and JavaScript, ensuring optimal performance and user experience</li>
                  <li>Worked with senior developers and designers to create convenient and visually appealing user interfaces, contributing to dynamic and interactive functions</li>
                  <li>Participated in debugging and solving web problems, ensuring optimal performance and cross-browser compatibility</li>
                  <li>Contributed to projects, supporting the development team and his feature in new research/technologies</li>
                  <li>[Optional] Achieved key successes such as completing a critical feature on time or improving page load speed/user metrics</li>
                </ul>
              </div>
            </div>
          </div>

          {/* Projects Section */}
          <div className="mb-10 print-section">
            <h2 className="text-3xl font-bold mb-6 pb-2 border-b-2 border-black">
              FEATURED PROJECTS
            </h2>
            <div className="space-y-6">
              {projects.slice(0, 4).map((project) => (
                <div key={project.id} className="border-l-4 border-black pl-6 hover:border-[#FFA500] transition-colors">
                  <div className="flex justify-between items-start mb-2">
                    <h3 className="text-lg font-bold">{project.title}</h3>
                    <span className="text-sm text-gray-600 whitespace-nowrap ml-4">{project.date}</span>
                  </div>
                  <p className="text-sm text-gray-700 mb-3 leading-relaxed">{project.description}</p>
                  <div className="flex flex-wrap gap-2 mb-3">
                    {project.technologies.slice(0, 5).map((tech, idx) => (
                      <span key={idx} className="px-3 py-1 border-2 border-black text-xs font-semibold rounded-lg hover:bg-black hover:text-white transition-colors">
                        {tech}
                      </span>
                    ))}
                  </div>
                  <div className="space-y-1 text-xs text-gray-700">
                    {project.features.slice(0, 3).map((feature, idx) => (
                      <p key={idx} className="flex items-start">
                        <span className="mr-2">▸</span>
                        <span>{feature}</span>
                      </p>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Contact Information */}
          <div className="mt-12 pt-8 border-t-2 border-black text-center print-section">
            <h3 className="text-2xl font-bold mb-4">GET IN TOUCH</h3>
            <div className="flex flex-wrap justify-center gap-6 text-sm">
              <a href="tel:+990906047174" className="hover:text-[#FFA500] transition-colors font-semibold">
                📱 +990 90 604 7174
              </a>
              <a href="mailto:shakhzodsoliyevmit@gmail.com" className="hover:text-[#FFA500] transition-colors font-semibold">
                📧 shakhzodsoliyevmit@gmail.com
              </a>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
