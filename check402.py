#!/usr/bin/env python3
import sys
import re

LOG_RE = re.compile(
    r'(?P<ip>\S+) \S+ \S+ \[(?P<ts>.*?)\] '
    r'"(?P<method>\S+) (?P<path>[^"]*) HTTP/[^"]+" '
    r'(?P<status>\d{3}) \S+ "[^"]*" "(?P<ua>[^"]*)"'
)

def main():
    if len(sys.argv) != 2:
        print("Usage: script.py /path/to/access.log", file=sys.stderr)
        sys.exit(1)

    path = sys.argv[1]

    per_ip_statuses = {}
    per_ip_uas = {}
    per_ip_count = {}

    with open(path, "r", encoding="utf8", errors="replace") as f:
        for line in f:
            m = LOG_RE.match(line)
            if not m:
                continue

            ip = m.group("ip")
            method = m.group("method")
            status = int(m.group("status"))
            ua = m.group("ua")

            if method != "GET":
                continue

            # ignore 3xx and 5xx
            if 300 <= status < 400 or 500 <= status < 600:
                continue

            per_ip_statuses.setdefault(ip, set()).add(status)
            per_ip_uas.setdefault(ip, set()).add(ua)
            per_ip_count[ip] = per_ip_count.get(ip, 0) + 1

    # Filter for IPs that exclusively have status 402
    candidates = [
        (ip, per_ip_count[ip], sorted(per_ip_uas[ip]))
        for ip, statuses in per_ip_statuses.items()
        if statuses == {402}
    ]

    # Sort by number of requests (descending)
    candidates.sort(key=lambda x: x[1], reverse=True)

    # Output
    for ip, count, uas in candidates:
        print(ip)
        print("  requests:", count)
        print("  user_agents:")
        for ua in uas:
            print("    " + ua)

if __name__ == "__main__":
    main()
