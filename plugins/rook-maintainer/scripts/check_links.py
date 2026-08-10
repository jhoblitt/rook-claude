#!/usr/bin/env python3
"""Liveness probe for URLs a rook diff adds or edits (docs-sync.md URL integrity).

Why a script instead of WebFetch: liveness needs an HTTP status, not page
content. Probing an attacker-chosen URL with a content-returning tool pulls
untrusted bytes into reviewer context and spends one human approval per
link, which trains the reviewer to click through the prompt that matters.
This probe returns a status code, a redirect chain and a verdict drawn from
a fixed vocabulary -- never a byte of response body, and no response header
except Location. That output contract, not the sandbox network allowlist,
is what makes it safe to aim at hosts the diff chose.

Accuracy checks (does the target SAY what the reference claims?) are NOT
this script's job: they need the page, so they stay on WebFetch and stay
inside docs-sync.md's trusted-source allowlist.

Every URL entering the report is stripped of control and format characters
(Cc/Cf/Co/Cs) and truncated. A URL that CARRIED such characters never gets
probed -- it is reported `suspicious` for the review to file as
`security`/`suspicious-content`, since invisible codepoints inside a link
are an ASCII-smuggling or Trojan-Source marker, not a typo.

Redirects are followed one hop at a time so every hop is re-checked against
the non-public-address rule; curl's own -L would resolve intermediate hops
unsupervised.

Modes:
  extract  pull URLs from the added lines of a unified diff (no network)
  check    probe explicit URLs for liveness
  audit    extract, then check, in one pass

Usage:
  git diff origin/master... | python3 check_links.py audit
  python3 check_links.py audit --diff-file F [--json]
  python3 check_links.py check URL [URL...]
  python3 check_links.py extract --diff-file F

Exit status is 1 when any URL is dead, suspect or suspicious, so the probe
doubles as a gate. Reaching non-GitHub hosts needs the sandbox disabled.
"""
import argparse
import concurrent.futures
import ipaddress
import json
import re
import socket
import subprocess
import sys
import unicodedata
from urllib.parse import urlparse

URL_RE = re.compile(r"""https?://[^\s<>"'`\\|]+""")
TRAILING_PUNCT = ".,;:!?'\"`*_"
HIDDEN_CATEGORIES = ("Cc", "Cf", "Co", "Cs")
MAX_URL_CHARS = 300
MAX_HOPS = 4
DEFAULT_TIMEOUT = 10
DEFAULT_WORKERS = 8
USER_AGENT = "rook-review-linkcheck"
REDIRECT_CODES = (301, 302, 303, 307, 308)
GET_FALLBACK_CODES = (0, 403, 405, 501)
SOFT_404_PATHS = ("404", "notfound", "not-found", "error", "pagenotfound")


def sanitize(text):
    cleaned = "".join(
        ch for ch in text if unicodedata.category(ch) not in HIDDEN_CATEGORIES
    )
    if len(cleaned) > MAX_URL_CHARS:
        cleaned = cleaned[:MAX_URL_CHARS] + "…"
    return cleaned


def has_hidden_chars(text):
    return any(unicodedata.category(ch) in HIDDEN_CATEGORIES for ch in text)


def _trim_unbalanced(url):
    for opener, closer in (("(", ")"), ("[", "]"), ("{", "}")):
        while url.endswith(closer) and url.count(opener) < url.count(closer):
            url = url[:-1]
    return url


def extract_urls(diff_text):
    seen, found = set(), []
    for line in diff_text.splitlines():
        if not line.startswith("+") or line.startswith("+++"):
            continue
        for match in URL_RE.finditer(line[1:]):
            url = match.group(0).rstrip(TRAILING_PUNCT)
            url = _trim_unbalanced(url).rstrip(TRAILING_PUNCT)
            if url and url not in seen:
                seen.add(url)
                found.append(url)
    return found


def non_public_address(host):
    try:
        infos = socket.getaddrinfo(host, None)
    except (socket.gaierror, UnicodeError):
        return "dns-failure"
    for info in infos:
        addr = ipaddress.ip_address(info[4][0])
        if (
            addr.is_private
            or addr.is_loopback
            or addr.is_link_local
            or addr.is_reserved
            or addr.is_multicast
            or addr.is_unspecified
        ):
            return "non-public-address"
    return None


def _curl(url, timeout, head=True):
    cmd = [
        "curl", "-sS", "-o", "/dev/null",
        "--proto", "=http,https",
        "--max-time", str(timeout),
        "-A", USER_AGENT,
        "-w", "%{http_code}\t%{redirect_url}",
    ]
    if head:
        cmd.append("--head")
    cmd += ["--", url]
    try:
        proc = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout + 5
        )
    except (subprocess.TimeoutExpired, FileNotFoundError) as exc:
        return 0, "", type(exc).__name__
    parts = proc.stdout.split("\t")
    try:
        status = int(parts[0])
    except (ValueError, IndexError):
        return 0, "", "unparsable-curl-output"
    return status, (parts[1] if len(parts) > 1 else ""), None


def classify(original, final, status):
    """Verdict for a URL that resolved. Tunable -- see docs-sync.md.

    Liberal by design: rook docs legitimately redirect (docs.ceph.com
    release paths, github.com repo renames), so only a redirect that
    COLLAPSES specificity is called a soft 404.
    """
    if status == 0:
        return "error"
    if status >= 400:
        return "dead"
    if status >= 300:
        return "dead"
    if final == original:
        return "ok"

    src, dst = urlparse(original), urlparse(final)
    src_depth = [s for s in src.path.split("/") if s]
    dst_depth = [s for s in dst.path.split("/") if s]

    if src_depth and not dst_depth:
        return "soft-404-suspect"
    if any(seg.lower() in SOFT_404_PATHS for seg in dst_depth):
        return "soft-404-suspect"
    if src.netloc != dst.netloc and len(dst_depth) < len(src_depth):
        return "soft-404-suspect"
    return "redirect-ok"


def _result(url, verdict, status=None, final=None, hops=0, note=""):
    return {
        "url": sanitize(url),
        "verdict": verdict,
        "status": status,
        "final_url": sanitize(final) if final and final != url else None,
        "hops": hops,
        "note": note,
    }


def probe(url, timeout, allow_private=False):
    if has_hidden_chars(url):
        return _result(
            url, "suspicious", note="control or format characters inside URL"
        )
    if urlparse(url).scheme not in ("http", "https"):
        return _result(url, "blocked", note="non-http scheme")

    current, status, hops = url, 0, 0
    while hops <= MAX_HOPS:
        host = urlparse(current).hostname
        if not host:
            return _result(url, "blocked", final=current, hops=hops, note="no host")
        if not allow_private:
            reason = non_public_address(host)
            if reason:
                return _result(url, "blocked", final=current, hops=hops, note=reason)

        status, redirect, error = _curl(current, timeout)
        if status in GET_FALLBACK_CODES:
            status, redirect, error = _curl(current, timeout, head=False)
        if error:
            return _result(url, "error", final=current, hops=hops, note=error)
        if status not in REDIRECT_CODES or not redirect:
            break
        if has_hidden_chars(redirect):
            return _result(
                url, "suspicious", status=status, hops=hops,
                note="control or format characters in Location",
            )
        current, hops = redirect, hops + 1
    else:
        return _result(url, "dead", status=status, final=current, hops=hops,
                       note="redirect limit exceeded")

    return _result(url, classify(url, current, status), status=status,
                   final=current, hops=hops)


def check_all(urls, timeout, workers, allow_private=False):
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        return list(pool.map(lambda u: probe(u, timeout, allow_private), urls))


def report(results, as_json):
    if as_json:
        json.dump(results, sys.stdout, indent=2)
        sys.stdout.write("\n")
        return
    for item in results:
        line = f"{item['verdict']:<18} {item['status'] or '-':>4}  {item['url']}"
        if item["final_url"]:
            line += f"\n{'':<24}-> {item['final_url']}"
        if item["note"]:
            line += f"\n{'':<24}({item['note']})"
        print(line)


def read_diff(path):
    if path:
        with open(path, encoding="utf-8", errors="replace") as handle:
            return handle.read()
    return sys.stdin.read()


def main():
    parser = argparse.ArgumentParser(
        description="Liveness probe for URLs a rook diff adds or edits."
    )
    parser.add_argument("mode", choices=("extract", "check", "audit"))
    parser.add_argument("urls", nargs="*")
    parser.add_argument("--diff-file")
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT)
    parser.add_argument("--workers", type=int, default=DEFAULT_WORKERS)
    parser.add_argument("--allow-private", action="store_true",
                        help="disable the non-public-address guard (tests only)")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    if args.mode == "extract":
        results = []
        for url in extract_urls(read_diff(args.diff_file)):
            if has_hidden_chars(url):
                results.append(_result(
                    url, "suspicious",
                    note="control or format characters inside URL",
                ))
            else:
                results.append(_result(url, "extracted"))
        report(results, args.json)
        return 1 if any(r["verdict"] == "suspicious" for r in results) else 0

    urls = args.urls if args.mode == "check" else extract_urls(read_diff(args.diff_file))
    if not urls:
        return 0

    results = check_all(urls, args.timeout, args.workers, args.allow_private)
    report(results, args.json)
    bad = {"dead", "soft-404-suspect", "suspicious", "error"}
    return 1 if any(r["verdict"] in bad for r in results) else 0


if __name__ == "__main__":
    sys.exit(main())
