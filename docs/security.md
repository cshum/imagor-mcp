# Security

**A trusted client of imagor, not a new surface on it.** It holds the signing secret to
mint valid URLs, like any backend. **Trust the component; do not trust the agent using
it.**

- **The secret is environment-only.** It is read from `IMAGOR_SECRET` and never crosses
  the tool surface: no tool takes a secret, and none returns one. An agent cannot leak
  what it was never given, and a prompt-injected one cannot be talked into signing with a
  key it does not hold.
- **Unsafe mode is opt-in and loud** — a warning on stderr, reported by `describe_imagor`.
- **Blast radius stays imagor's to set.** A URL from an errant agent is still subject to
  every limit on the instance it lands on: `HTTP_LOADER_ALLOWED_SOURCES`,
  `VIPS_MAX_RESOLUTION`, `IMAGOR_REQUEST_TIMEOUT`, and the `expire()` filter for a hard
  expiry. All of them are imagor's settings, documented at
  **[docs.imagor.net/security](https://docs.imagor.net/security)** — this server has no
  opinion about them and no way to override them.

Do not host this multi-tenant for users whose keys you would then hold.
