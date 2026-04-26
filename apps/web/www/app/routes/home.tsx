import { Navbar } from '@/components/shared/navbar';
import { HomePage } from '@/features/home/home';

export default function Home() {
  return (
    <div className="relative min-h-screen bg-background text-white overflow-hidden flex flex-col antialised">
      <div className="fixed top-0 left-0 right-0 z-50 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 pointer-events-none">
        <div className="pointer-events-auto">
          <Navbar />
        </div>
      </div>
      
      <main className="flex-1 relative w-full h-full bg-background">
        <HomePage />
      </main>
    </div>
  );
}
