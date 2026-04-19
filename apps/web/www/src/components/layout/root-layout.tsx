import React from 'react';
import { Outlet } from 'react-router-dom';
import { Navbar } from '@/components/shared/navbar';

export const RootLayout: React.FC = () => {
  return (
    <div className="relative min-h-screen bg-black text-white overflow-hidden flex flex-col antialised">
      {/* Navbar Container */}
      <div className="fixed top-0 left-0 right-0 z-50 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 pointer-events-none">
        {/* Enable pointer events on Navbar itself while keeping wrapper pointer-events-none so we can interact with background */}
        <div className="pointer-events-auto">
          <Navbar />
        </div>
      </div>
      
      {/* Page Content */}
      <main className="flex-1 relative w-full h-full">
        <Outlet />
      </main>
    </div>
  );
};
