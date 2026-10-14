# Configuration

**Set exactly one of `IMAGOR_SECRET` or `IMAGOR_UNSAFE`.** Setting both, or neither, is
refused at startup.

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects **signed** mode: `<signature>/<path>`. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects **unsafe** mode: `unsafe/<path>`, development only. Must be explicit. |
| `IMAGOR_BASE_URL` | Your imagor origin, as **you** reach it. Needed for `verify_imagor_url`, and it adds `absolute_url` to output. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256` or `sha512`. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters. `0` means none. |
| `IMAGOR_MCP_EXTRA_FILTERS` | Extra filter names for forks that register their own. |

The signer settings must match your imagor exactly, or it rejects the URLs.

**Compose in `IMAGOR_UNSAFE`; serve in `IMAGOR_SECRET`.** A signature covers the whole path,
so editing any part of a signed URL invalidates it. Switch when the URL is finished and
something else is going to serve it.

**`IMAGOR_BASE_URL` does two jobs.** It is where the server fetches a render back to verify
it, and it is what the URLs it returns are prefixed with. Both have to be an address *you*
can open, which is why it names `localhost` above and not a container-internal host.

That is the one real cost of running the server in a container. There, the host is
`host.docker.internal` and `localhost` is the container, so the server needs
`-e IMAGOR_BASE_URL=http://host.docker.internal:8000` to work at all — and every URL it hands
you then carries a name your browser cannot resolve. On the host, one address does both
jobs.

## Making previews faster

`verify_imagor_url` fetches through imagor's `preview()` filter, which skips result
storage and opts into the in-memory cache. **Turn the cache on** and repeated looks
at the same composition get much cheaper — measured on a 1600×600 composition,
**386 ms** for the first preview and **25 ms** for a repeat:

```bash
VIPS_CACHE_SIZE=52428800      # byte budget; 0 disables the cache (default)
VIPS_CACHE_MAX_WIDTH=2400     # images larger than this are not cached
VIPS_CACHE_MAX_HEIGHT=2000
```


Running the server in a container changes `IMAGOR_BASE_URL` — see the paragraph above — and so
does anything that puts a proxy or a different hostname in front of imagor.
