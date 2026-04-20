/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly MAIN_VITE_MILDSTACK_EXECUTABLE: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
