# Lookup RewriteMap Helper

This program implements an Apache `RewriteMap` helper that can serve many independent allow lists from a single long‑running process. It reads lookups from stdin (one per line) and replies on stdout with `FOUND` or `NULL`, as required by the RewriteMap protocol.

## How it works

- Input format: each line must be `<configFilename>;<lookupValue>`, separated by a semicolon. The lookup value may contain spaces.
- Config selection: `configFilename` must match `name[.(ri|re|net)].list`. The `name` must be alphanumeric. The optional suffix chooses the match mode:
  - `.ri.list` → case-insensitive regex
  - `.re.list` → case-sensitive regex
  - `.net.list` → IP address and CIDR matching
  - `.list` (no suffix) → exact literal matching
- Location: config files are loaded from the same directory as the binary.
- Caching: configs are cached by filename; the first request for a given config loads it, and subsequent requests reuse it. The file’s mtime is checked on each lookup; if it changes (or the file appears/disappears), the checker is rebuilt.
- Matching:
  - Literal: exact string equality
  - Regex: compiled per pattern; invalid patterns are skipped
  - Nets: supports exact IPs and CIDR blocks; invalid entries are skipped
- Empty or missing configs: return `NULL` for all lookups.

## Usage with Apache

Example `RewriteMap` configuration:

```
RewriteMap lookup prg:/path/to/lookup
```

Example rules that supply the config filename and lookup value:

```
# IP allow list (CIDR-aware)
RewriteCond ${lookup:addresses.net.list %{REMOTE_ADDR}|NOT_FOUND} !=NOT_FOUND

# User-Agent allow list (case-insensitive regex)
RewriteCond ${lookup:useragents.ri.list %{HTTP_USER_AGENT}|NOT_FOUND} !=NOT_FOUND
```

Each map invocation sends `"<config>;<lookup>"` to the helper. The helper responds with `FOUND` if the lookup matches the chosen config, otherwise `NULL`.
