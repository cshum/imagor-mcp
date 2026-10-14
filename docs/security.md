# Security

**A trusted client of imagor, not a new surface on it.** It holds the signing secret to
mint valid URLs, like any backend. **Trust the component; do not trust the agent using
it.**

- **Unsafe mode is opt-in and loud** — a warning on stderr, reported by `describe_imagor`.
- **Blast radius stays imagor's to set** — `HTTP_LOADER_ALLOWED_SOURCES`,
  `VIPS_MAX_RESOLUTION`, `IMAGOR_REQUEST_TIMEOUT`. A URL from an errant agent is still
  subject to every one.

Do not host this multi-tenant for users whose keys you would then hold.

