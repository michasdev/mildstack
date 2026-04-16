import electronLogo from '@renderer/assets/electron.svg'
import Versions from '@renderer/components/Versions'
import { Button } from '@renderer/components/ui/button'

function AppShell(): React.JSX.Element {
  const ipcHandle = (): void => window.electron.ipcRenderer.send('ping')

  return (
    <main className="min-h-screen w-full overflow-hidden bg-[radial-gradient(circle_at_top,_rgba(120,119,198,0.18),_transparent_40%),linear-gradient(180deg,_#0b1020_0%,_#060816_100%)] px-6 py-10 text-white">
      <section className="mx-auto flex w-full max-w-5xl flex-col gap-8 rounded-[2rem] border border-white/10 bg-white/5 p-8 shadow-2xl shadow-black/30 backdrop-blur-xl md:p-12">
        <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div className="max-w-xl space-y-4">
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-cyan-300/80">
              Powered by MildStack
            </p>
            <h1 className="text-4xl font-semibold tracking-tight text-white md:text-6xl">
              Inspect local AWS resources with a calmer desktop shell.
            </h1>
            <p className="max-w-lg text-base leading-7 text-slate-300 md:text-lg">
              MildStack App gives us a focused workspace for future S3, DynamoDB, and service
              catalog explorers while keeping native communication behind Electron.
            </p>
          </div>

          <div className="flex items-center justify-center">
            <img
              alt="MildStack App logo"
              className="h-28 w-28 drop-shadow-[0_12px_40px_rgba(34,211,238,0.25)] md:h-36 md:w-36"
              src={electronLogo}
            />
          </div>
        </div>

        <div className="flex flex-col gap-4 sm:flex-row">
          <Button onClick={ipcHandle} className="shadow-lg shadow-cyan-500/20">
            Send IPC
          </Button>
          <Button
            variant="outline"
            onClick={() => window.open('https://electron-vite.org/', '_blank', 'noreferrer')}
          >
            Documentation
          </Button>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
            <p className="text-sm font-medium text-slate-200">Electron boundary</p>
            <p className="mt-2 text-sm leading-6 text-slate-400">
              Renderer UI stays browser-safe while native APIs remain behind preload and main.
            </p>
          </div>
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
            <p className="text-sm font-medium text-slate-200">Tailwind v4</p>
            <p className="mt-2 text-sm leading-6 text-slate-400">
              Utility classes now shape the layout directly, so future screens stay consistent.
            </p>
          </div>
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
            <p className="text-sm font-medium text-slate-200">CossUI ready</p>
            <p className="mt-2 text-sm leading-6 text-slate-400">
              Button primitives are in place for the app shell and upcoming explorer workflows.
            </p>
          </div>
        </div>

        <Versions />
      </section>
    </main>
  )
}

export default AppShell
