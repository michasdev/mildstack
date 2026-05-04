import mildstack from '@/assets/logos/mildstack-logo-full-white.png';
import { Download } from 'lucide-react';

export function Footer() {
    const year = new Date().getFullYear();

    const website = [
        {
            title: 'Home',
            href: '/#home',
        },
        {
            title: 'Get Started',
            href: '/#get-started',
        },
        {
            title: 'Desktop App',
            href: '/#desktop-app',
        },
        {
            title: 'Docs',
            href: '/docs',
        }
    ];

    const socialLinks = [
        {
            icon: <svg
                className="size-5"
                fill="white"
                viewBox="0 0 24 24"
                xmlns="http://w3.org"
            >
                <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
            </svg>,
            link: 'https://github.com/michasdev/mildstack',
        },
    ];
    return (
        <footer className="relative">
            <div className="bg-[radial-gradient(35%_80%_at_30%_0%,--theme(--color-primary/.1),transparent)] mx-auto max-w-5xl md:border-x">
                <div className="bg-border absolute inset-x-0 h-px w-full" />
                <div className="grid max-w-7xl grid-cols-5 gap-6 p-4">
                    <div className="col-span-6 flex flex-col gap-3 md:col-span-4">
                        <a href="/#home" className="w-max">
                            <img src={mildstack} alt="MildStack Logo" className="h-12" />
                        </a>
                        <p className="w-full text-muted-foreground max-w-xl font-mono text-sm text-balance">
                            Mildstack: Run AWS services locally. Free, lightweight, and open-source.
                            <br />
                            The definitive LocalStack alternative.
                        </p>
                        <div className="flex gap-2">
                            {socialLinks.map((item, i) => (
                                <a
                                    key={i}
                                    className="hover:bg-accent rounded-md border p-1.5 flex flex-row items-center gap-2 text-sm font-light"
                                    target="_blank"
                                    href={item.link}
                                >
                                    {item.icon} Github
                                </a>
                            ))}
                            <a
                                className="hover:bg-accent rounded-md border p-1.5 flex flex-row items-center gap-2 text-sm font-light"
                                href="/download"
                            >
                                <Download className="size-5" />
                                Download App
                            </a>
                        </div>
                    </div>
                    <div className="col-span-3 w-full md:col-span-1">
                        <span className="text-muted-foreground mb-1 text-xs">
                            Resources
                        </span>
                        <div className="flex flex-col gap-1">
                            {website.map(({ href, title }, i) => (
                                <a
                                    key={i}
                                    className={`w-max py-1 text-sm duration-200 hover:underline`}
                                    href={href}
                                >
                                    {title}
                                </a>
                            ))}
                        </div>
                    </div>
                </div>
                <div className="bg-border absolute inset-x-0 h-px w-full" />
                <div className="flex max-w-4xl flex-col justify-between gap-2 pt-2 pb-5 mx-auto">
                    <p className="text-muted-foreground text-center text-sm font-light">
                        © <a href="https://mildstack.dev">MildStack {year}</a>. GPL-3.0 licensed. <a href="https://github.com/michasdev/mildstack" target="_blank" rel="noopener noreferrer">View Repository.</a>
                    </p>
                </div>
            </div>
        </footer>
    );
}
