/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Sentry DSN for the desktop UI (separate project from the Rust core). Empty ⇒ disabled. */
  readonly VITE_SENTRY_DSN?: string;
  /** "1" enables update checks. Off until the updater URL + signing key are our own. */
  readonly VITE_ENABLE_UPDATER?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
