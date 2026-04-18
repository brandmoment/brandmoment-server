export async function register() {
  // Polyfill localStorage for SSR — better-auth/react accesses it during module init
  if (typeof globalThis.localStorage === "undefined") {
    const storage: Record<string, string> = {};
    (globalThis as Record<string, unknown>).localStorage = {
      getItem: (key: string) => storage[key] ?? null,
      setItem: (key: string, value: string) => {
        storage[key] = value;
      },
      removeItem: (key: string) => {
        delete storage[key];
      },
      clear: () => {
        Object.keys(storage).forEach((k) => delete storage[k]);
      },
      get length() {
        return Object.keys(storage).length;
      },
      key: (index: number) => Object.keys(storage)[index] ?? null,
    };
  }
}
