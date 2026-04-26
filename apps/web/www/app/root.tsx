import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from 'react-router';
import { RootProvider } from 'fumadocs-ui/provider/react-router';
import type { Route } from './+types/root';
import './app.css';
import SearchDialog from '@/components/search';
import NotFound from './routes/not-found';

export const links: Route.LinksFunction = () => [
  { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
  {
    rel: 'preconnect',
    href: 'https://fonts.gstatic.com',
    crossOrigin: 'anonymous',
  },
  {
    rel: 'stylesheet',
    href: 'https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&display=swap',
  },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" type="image/svg+xml" href="/mildstack-logo.svg" />
        <title>MildStack: Free Open Source LocalStack Alternative</title>
        <meta name="description" content="MildStack is a lightweight, local-first AWS emulator and LocalStack alternative. Build and test S3, DynamoDB, Lambda, SQS, and more locally with zero cloud costs and minimal overhead." />
        <meta name="keywords" content="aws, cloud, localstack, localstack alternative, aws emulator, local aws development, s3 local, dynamodb local, lambda local, sqs local, cloud development tools, s3, sqs, dynamodb, lambda, local mildstack, mildstack, open source aws emulator, ministack" />

        <meta property="og:type" content="website" />
        <meta property="og:url" content="https://mildstack.dev/" />
        <meta property="og:title" content="MildStack - Free Open Source LocalStack Alternative" />
        <meta property="og:description" content="Lightweight, local-first AWS emulator for developers. Fast, simple, and memory-efficient local cloud API." />
        <meta property="og:image" content="/og-image.png" />

        <meta property="twitter:card" content="summary_large_image" />
        <meta property="twitter:url" content="https://mildstack.dev/" />
        <meta property="twitter:title" content="MildStack - Open Source LocalStack Alternative" />
        <meta property="twitter:description" content="Lightweight, local-first AWS emulator for developers. Fast, simple, and memory-efficient local cloud API." />
        <meta property="twitter:image" content="/og-image.png" />
        <Meta />
        <Links />
      </head>
      <body className="flex flex-col min-h-screen">
        <RootProvider search={{ SearchDialog }}>{children}</RootProvider>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  return <Outlet />;
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = 'Oops!';
  let details = 'An unexpected error occurred.';
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) return <NotFound />;
    message = 'Error';
    details = error.statusText;
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="pt-16 p-4 w-full max-w-[1400px] mx-auto">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full p-4 overflow-x-auto">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  );
}
