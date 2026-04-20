import {
	Grid2X2Plus,
} from 'lucide-react';
import mildstack from '@/assets/logos/mildstack-logo-full-white.png';

export function Footer() {
	const year = new Date().getFullYear();

	const website = [
		{
			title: 'Home',
			href: '#',
		},
		{
			title: 'Features',
			href: '#',
		},
		{
			title: 'Services',
			href: '#',
		},
		{
			title: 'Docs',
			href: '#',
		}
	];

	const resources = [
		{
			title: 'Blog',
			href: '#',
		},
		{
			title: 'Help Center',
			href: '#',
		},
		{
			title: 'Contact Support',
			href: '#',
		},
		{
			title: 'Community',
			href: '#',
		},
		{
			title: 'Security',
			href: '#',
		},
	];

	const socialLinks = [
		{
			icon: <Grid2X2Plus className="size-4" />,
			link: 'https://github.com/michasdev/mildstack',
		},
	];
	return (
		<footer className="relative">
			<div className="bg-[radial-gradient(35%_80%_at_30%_0%,--theme(--color-primary/.1),transparent)] mx-auto max-w-5xl md:border-x">
				<div className="bg-border absolute inset-x-0 h-px w-full" />
				<div className="grid max-w-7xl grid-cols-5 gap-6 p-4">
					<div className="col-span-6 flex flex-col gap-3 md:col-span-4">
						<a href="#" className="w-max">
							<img src={mildstack} alt="MildStack Logo" className="h-12" />
						</a>
						<p className="w-full text-muted-foreground max-w-xl font-mono text-sm text-balance">
							Mildstack: Run AWS services locally. Free, lightweight, and open-source.
                            <br/>
                            The definitive LocalStack alternative.
						</p>
						<div className="flex gap-2">
							{socialLinks.map((item, i) => (
								<a
									key={i}
									className="hover:bg-accent rounded-md border p-1.5"
									target="_blank"
									href={item.link}
								>
									{item.icon}
								</a>
							))}
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
				<div className="flex max-w-4xl flex-col justify-between gap-2 pt-2 pb-5">
					<p className="text-muted-foreground text-center text-sm font-light">
						© <a href="https://mildstack.dev">MildStack {year}</a>. MIT Licensed. <a href="https://github.com/michasdev/mildstack" target="_blank" rel="noopener noreferrer">View Repository.</a>
					</p>
				</div>
			</div>
		</footer>
	);
}