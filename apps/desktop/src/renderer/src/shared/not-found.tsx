import { Link } from 'react-router'

function NotFoundPage(): React.JSX.Element {
  return (
    <div className="flex flex-col gap-4 p-6">
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <p className="text-sm text-muted-foreground">
        The page you are looking for does not exist.
      </p>
      <Link className="text-sm underline underline-offset-4" to="/">
        Go home
      </Link>
    </div>
  )
}

export default NotFoundPage
