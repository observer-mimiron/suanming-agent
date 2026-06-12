// ---------------------------------------------------------------------------
// In-process file-level write lock
// ---------------------------------------------------------------------------
//
// Prevents TOCTOU races on shared wiki files (index.md, log.md) when multiple
// concurrent requests (e.g. two browser tabs ingesting at the same time) hit
// the same Next.js server process.
//
// Design: a Map of promise chains keyed by an arbitrary string (typically a
// file path or logical resource name). Each call to `withFileLock(key, fn)`
// chains `fn` after the previous promise for that key, guaranteeing serial
// execution per key while allowing unrelated keys to proceed in parallel.
//
// Limitations:
// - In-process only — does not protect against multiple server processes
//   (which would require OS-level lockfiles). Next.js dev and single-instance
//   production deployments run a single process, so this covers the common case.
// - The map grows one entry per unique key and is never pruned. In practice
//   the wiki has very few shared files (index.md, log.md, cross-ref), so this
//   is not a concern.
// ---------------------------------------------------------------------------

const locks = new Map<string, Promise<unknown>>();

/**
 * Execute `fn` while holding an in-process lock for `key`.
 *
 * If another call with the same key is already in flight, `fn` will wait
 * until that call settles (resolves or rejects) before starting. Calls
 * with different keys run concurrently.
 *
 * @param key  Logical lock name — typically a file path or resource id.
 * @param fn   Async function to execute under the lock.
 * @returns    The resolved value of `fn`.
 * @throws     Re-throws whatever `fn` throws, after releasing the lock.
 */
export async function withFileLock<T>(
  key: string,
  fn: () => Promise<T>,
): Promise<T> {
  const prev = locks.get(key) ?? Promise.resolve();

  // Chain fn after the previous holder. `then(fn, fn)` ensures fn runs
  // regardless of whether the previous holder resolved or rejected.
  const next = prev.then(fn, fn);

  // Store a silenced version on the chain so an unhandled-rejection
  // warning is never triggered on the *chain itself* (callers still get
  // the real rejection via the `next` they await).
  locks.set(key, next.catch(() => {}));

  return next;
}

/**
 * Reset all locks. **Test-only** — exported so tests can start with a clean
 * slate without leaking state between test files.
 */
export function _resetLocks(): void {
  locks.clear();
}
