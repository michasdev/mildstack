import React, { useState } from 'react';
import { motion, AnimatePresence, useScroll, useMotionValueEvent } from 'motion/react';
import { Star, Download } from 'lucide-react';
import logoFullWhite from '@/assets/logos/mildstack-logo-full-white.svg'

export const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const { scrollY } = useScroll();

  useMotionValueEvent(scrollY, "change", (latest) => {
    if (latest > 50) {
      setScrolled(true);
    } else {
      setScrolled(false);
    }
  });

  return (
    <>
      <motion.div
        initial={{ y: -20, opacity: 0 }}
        animate={{ 
          y: 0, 
          opacity: 1,
          scale: scrolled ? 0.98 : 1,
        }}
        transition={{ duration: 0.5 }}
        className={`px-4 backdrop-blur-3xl rounded-full py-4 flex justify-between items-center transition-all duration-300 ${
          scrolled ? 'bg-black/80 border border-white/10 shadow-2xl shadow-white/5' : 'bg-black/50'
        }`}
      >
        <div className="flex items-center">
          <div className="text-2xl font-bold">
            <img src={logoFullWhite} className='h-10 w-auto' alt="Logo" />
          </div>
          <div className="hidden md:flex items-center space-x-6 ml-8">
            <button className="px-4 py-2 bg-gray-800/50 hover:bg-gray-700/50 rounded-full text-sm transition-colors">Start</button>
            <button className="px-4 py-2 text-sm hover:text-gray-300 transition-colors">Features</button>
            <button className="px-4 py-2 text-sm hover:text-gray-300 transition-colors">Services</button>
            <button className="px-4 py-2 text-sm hover:text-gray-300 transition-colors">Docs</button>
            <button className="px-4 py-2 text-sm hover:text-gray-300 transition-colors">Community</button>
          </div>
        </div>
        <div className="flex items-center space-x-4">
          <button className="hidden md:flex items-center gap-2 px-4 py-2 text-sm hover:text-gray-300 transition-colors">
            <Download className="size-4" />
            Download App
          </button>
          <a 
            href="https://github.com/michasdev/mildstack" 
            target="_blank" 
            rel="noopener noreferrer"
            className="flex items-center gap-2 px-4 py-2 bg-white/10 backdrop-blur-sm rounded-full text-sm hover:bg-white/20 transition-colors border border-white/10"
          >
            <Star className="size-4 fill-yellow-400 text-yellow-400" />
            Give a Star
          </a>
          {/* Mobile menu button */}
          <button
            className="md:hidden p-2 rounded-md focus:outline-none"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          >
            {mobileMenuOpen ? (
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            ) : (
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            )}
          </button>
        </div>
      </motion.div>

      {/* Mobile menu */}
      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="md:hidden fixed inset-0 z-50 bg-black/95 backdrop-blur-lg z-9999">
            <div className="flex flex-col items-center justify-center h-full space-y-6 text-lg">
              <button
                className="absolute top-6 right-6 p-2"
                onClick={() => setMobileMenuOpen(false)}
              >
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
              <button className="px-6 py-3 bg-gray-800/50 rounded-full">Start</button>
              <button className="px-6 py-3">Features</button>
              <button className="px-6 py-3">Services</button>
              <button className="px-6 py-3">Docs</button>
              <button className="px-6 py-3">Community</button>
              <button className="px-6 py-3 flex items-center gap-2">
                <Download className="size-5" />
                Download App
              </button>
              <a 
                href="https://github.com/michasdev/mildstack"
                target="_blank"
                rel="noopener noreferrer"
                className="px-6 py-3 bg-white/10 backdrop-blur-sm rounded-full flex items-center gap-2"
              >
                <Star className="size-5 fill-yellow-400 text-yellow-400" />
                Give a Star
              </a>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
};
