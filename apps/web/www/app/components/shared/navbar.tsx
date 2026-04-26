import React, { useState } from 'react';
import { motion, AnimatePresence, useScroll, useMotionValueEvent } from 'motion/react';
import { Star, Download, Menu, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import logoFullWhite from '../../assets/logos/mildstack-logo-full-white.png'

const NAV_LINKS = [
  { label: 'Home', variant: 'outline' },
  { label: 'Features', variant: 'ghost' },
  { label: 'Services', variant: 'ghost' },
  { label: 'Docs', variant: 'ghost' },
  { label: 'Community', variant: 'ghost' },
] as const;

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

  const NavLinks = ({ isMobile = false, closeMenu }: { isMobile?: boolean; closeMenu?: () => void }) => (
    <>
      {NAV_LINKS.map((link) => (
        <Button
          key={link.label}
          variant={isMobile && link.label === 'Home' ? 'secondary' : link.variant}
          className={cn(
            'rounded-full transition-all',
            isMobile ? 'px-8 h-12 text-lg w-full max-w-[280px]' : 'h-9 px-4 text-sm hover:text-gray-300'
          )}
          onClick={closeMenu}
        >
          {link.label}
        </Button>
      ))}
    </>
  );

  const ActionButtons = ({ isMobile = false, closeMenu }: { isMobile?: boolean; closeMenu?: () => void }) => (
    <>
      <Button
        variant="ghost"
        className={cn(
          'items-center gap-2 rounded-full transition-colors',
          isMobile ? 'px-8 h-12 text-lg w-full max-w-[280px] flex' : 'hidden md:flex h-9 px-4 text-sm hover:text-gray-300'
        )}
        onClick={closeMenu}
      >
        <Download className={isMobile ? 'size-5' : 'size-4'} />
        Download App
      </Button>
      <a
        href="https://github.com/michasdev/mildstack"
        target="_blank"
        rel="noopener noreferrer"
        onClick={closeMenu}
        className={cn(
          'items-center gap-2 backdrop-blur-sm rounded-full transition-colors border border-purple-500/20 bg-gradient-to-br from-purple-500/20 to-transparent',
          isMobile ? 'px-8 py-3 text-lg w-full max-w-[280px] justify-center flex' : 'hidden md:flex px-4 py-2 text-sm hover:bg-white/20'
        )}
      >
        <Star className={cn('fill-yellow-400 text-yellow-400', isMobile ? 'size-5' : 'size-4')} />
        Give a Star
      </a>
    </>
  );

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
        className={cn(
          "px-4 backdrop-blur-3xl rounded-full py-4 flex justify-between items-center transition-all duration-300",
          scrolled ? 'bg-background/80 border shadow-2xl shadow-white/5' : 'bg-background/50'
        )}
      >
        <div className="flex items-center">
          <div className="text-2xl font-bold">
            <img src={logoFullWhite} className='h-9 w-auto' alt="Logo" />
          </div>
          <div className="hidden md:flex items-center space-x-2 ml-8">
            <NavLinks />
          </div>
        </div>
        <div className="flex items-center space-x-4">
          <ActionButtons />
          
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          >
            {mobileMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </Button>
        </div>
      </motion.div>

      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="md:hidden fixed inset-0 z-50 bg-background/95 backdrop-blur-lg">
            <div className="flex flex-col items-center justify-center h-full space-y-4 px-6">
              <Button
                variant="ghost"
                size="icon"
                className="absolute top-6 right-6"
                onClick={() => setMobileMenuOpen(false)}
              >
                <X className="h-6 w-6" />
              </Button>
              <NavLinks isMobile closeMenu={() => setMobileMenuOpen(false)} />
              <ActionButtons isMobile closeMenu={() => setMobileMenuOpen(false)} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
};
