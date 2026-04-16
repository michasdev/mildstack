// Package runtime owns the in-memory application runtime ledger for the shared binary.
//
// Bootstrap registers routes explicitly once through composition, then the
// manager records metadata and ports behind a mutex. Snapshot and Ports return
// sorted copies so delivery layers can render stable output without mutating
// internal state. Shared service state uses the same clone-on-read and
// clone-on-write discipline through MemoryStateHook.
package runtime
