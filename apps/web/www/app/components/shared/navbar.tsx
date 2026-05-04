import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence, useScroll, useMotionValueEvent } from 'motion/react';
import { Star, Download, Menu, X } from 'lucide-react';
import { useLocation } from 'react-router';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import logoFullWhite from '../../assets/logos/mildstack-logo-full-white.png'

type NavLink = {
  label: string;
  variant: 'outline' | 'ghost';
  href: string;
  sectionId?: string;
};

const NAV_LINKS: NavLink[] = [
  { label: 'Home', variant: 'ghost', href: '/#home', sectionId: 'home' },
  { label: 'Get Started', variant: 'ghost', href: '/#get-started', sectionId: 'get-started' },
  { label: 'Desktop App', variant: 'ghost', href: '/#desktop-app', sectionId: 'desktop-app' },
  { label: 'FAQ', variant: 'ghost', href: '/#faq', sectionId: 'faq' },
  { label: 'Docs', variant: 'ghost', href: '/docs' },
];

const LANDING_SECTION_IDS = ['home', 'get-started', 'desktop-app', 'faq'] as const;
const ACTIVE_NAV_ITEM_CLASS =
  'border border-purple-500/20 bg-gradient-to-br from-purple-500/20 to-transparent backdrop-blur-sm';
type LandingSectionId = (typeof LANDING_SECTION_IDS)[number];
type SectionEntry = {
  sectionId: LandingSectionId;
  top: number;
};

export const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [activeSection, setActiveSection] = useState<string | null>('home');
  const { scrollY } = useScroll();
  const { pathname } = useLocation();

  const handleSectionClick = (
    event: React.MouseEvent<HTMLAnchorElement>,
    sectionId: string,
    closeMenu?: () => void,
  ) => {
    const target = document.getElementById(sectionId);
    if (!target) {
      closeMenu?.();
      return;
    }

    event.preventDefault();
    const navbarOffset = 112;
    const targetTop = target.getBoundingClientRect().top + window.scrollY - navbarOffset;

    window.scrollTo({
      top: Math.max(targetTop, 0),
      behavior: 'smooth',
    });

    setActiveSection(sectionId);
    closeMenu?.();
  };

  useMotionValueEvent(scrollY, "change", (latest) => {
    if (latest > 50) {
      setScrolled(true);
    } else {
      setScrolled(false);
    }
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const resolveActiveSection = () => {
      const activationOffset = 140;
      const sectionEntries: SectionEntry[] = LANDING_SECTION_IDS.flatMap((sectionId) => {
        const section = document.getElementById(sectionId);
        if (!section) return [];

        return [{
          sectionId,
          top: section.getBoundingClientRect().top + window.scrollY,
        }];
      });

      if (sectionEntries.length === 0) {
        setActiveSection(null);
        return;
      }

      if (window.scrollY <= 8) {
        setActiveSection('home');
        return;
      }

      const activationPosition = window.scrollY + activationOffset;

      const sortedEntries = [...sectionEntries].sort((a, b) => a.top - b.top);
      let nextSection: LandingSectionId = sortedEntries[0].sectionId;

      for (let index = 0; index < sortedEntries.length; index += 1) {
        const currentEntry = sortedEntries[index];
        const nextEntry = sortedEntries[index + 1];
        const currentStart = currentEntry.top;
        const currentEnd = nextEntry ? nextEntry.top : Number.POSITIVE_INFINITY;

        if (activationPosition >= currentStart && activationPosition < currentEnd) {
          nextSection = currentEntry.sectionId;
          break;
        }
      }

      setActiveSection((previous) => (previous === nextSection ? previous : nextSection));
    };

    resolveActiveSection();
    window.addEventListener('scroll', resolveActiveSection, { passive: true });
    window.addEventListener('resize', resolveActiveSection);

    return () => {
      window.removeEventListener('scroll', resolveActiveSection);
      window.removeEventListener('resize', resolveActiveSection);
    };
  }, [pathname]);

  const NavLinks = ({ isMobile = false, closeMenu }: { isMobile?: boolean; closeMenu?: () => void }) => (
    <>
      {NAV_LINKS.map((link) => {
        const isActive = Boolean(link.sectionId) && activeSection === link.sectionId;

        return (
          <Button
            asChild
            key={link.label}
            variant={isActive ? 'ghost' : link.variant}
            className={cn(
              'rounded-full transition-all',
              isActive && ACTIVE_NAV_ITEM_CLASS,
              isMobile ? 'px-8 h-12 text-lg w-full max-w-[280px]' : 'h-9 px-4 text-sm hover:text-gray-300'
            )}
          >
            <a
              href={link.href}
              onClick={(event) => {
                if (link.sectionId) {
                  handleSectionClick(event, link.sectionId, closeMenu);
                  return;
                }

                closeMenu?.();
              }}
            >
              {link.label}
            </a>
          </Button>
        );
      })}
    </>
  );

  const isDownloadActive = pathname === '/download';

  const ActionButtons = ({ isMobile = false, closeMenu }: { isMobile?: boolean; closeMenu?: () => void }) => (
    <>
      <Button
        asChild
        variant="ghost"
        className={cn(
          'items-center gap-2 rounded-full transition-colors',
          isDownloadActive && ACTIVE_NAV_ITEM_CLASS,
          isMobile ? 'px-8 h-12 text-lg w-full max-w-[280px] flex' : 'hidden md:flex h-9 px-4 text-sm hover:text-gray-300'
        )}
      >
        <a href="/download" onClick={closeMenu}>
          <Download className={isMobile ? 'size-5' : 'size-4'} />
          Download
        </a>
      </Button>
      <a
        href="https://github.com/michasdev/mildstack"
        target="_blank"
        rel="noopener noreferrer"
        onClick={closeMenu}
        className={cn(
          'items-center gap-2 backdrop-blur-sm rounded-full transition-colors border border-purple-500/20 bg-gradient-to-br from-purple-500/20 to-transparent hover:from-purple-600/20',
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
          "px-4 backdrop-blur-3xl rounded-full md:py-4 flex justify-between items-center transition-all duration-300",
          mobileMenuOpen ? 'py-4' : 'py-2.5',
          scrolled ? 'bg-background/80 border shadow-2xl shadow-white/5' : 'bg-background/50'
        )}
      >
        <div className="flex items-center">
          <div className="text-2xl font-bold">
            <a href="/" aria-label="MildStack home">
              <img
                src={logoFullWhite}
                className={cn('w-auto transition-all duration-300 md:h-9', mobileMenuOpen ? 'h-9' : 'h-8')}
                alt="Logo"
              />
            </a>
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
