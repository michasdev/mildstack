import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { appName, gitConfig } from './shared';
import logoWhite from "@/assets/logos/mildstack-logo-full-white.png"
import logoBlack from "@/assets/logos/mildstack-logo-full-black.png"
export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      // JSX supported
      title: (
      <>
        <img 
          src={logoWhite} 
          className='h-7 w-auto hidden dark:block'
          alt="MildStack Logo" 
        />
        <img 
          src={logoBlack} 
          className='h-7 w-auto block dark:hidden'
          alt="MildStack Logo" 
        />
      </>
    ),
    },
    githubUrl: `https://github.com/${gitConfig.user}/${gitConfig.repo}`,
  };
}
