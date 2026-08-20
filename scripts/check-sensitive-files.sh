#!/bin/sh
set -eu

tracked_sensitive_files="$(
  git ls-files |
    grep -E '(^|/)\.env$|(^|/)scan-darisini\.md$|(^|/)response\.json$|\.(pem|key|p12|pfx|pdf|xlsx|xls|db)$' ||
    true
)"

if [ -n "$tracked_sensitive_files" ]; then
  echo "Sensitive/generated files must not be tracked:"
  echo "$tracked_sensitive_files"
  exit 1
fi

content_hits="$(
  git grep -Il -E \
    '(-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----|__Secure-next-auth\.session-token=[A-Za-z0-9._-]{32,}|(SB-)?Mid-server-[A-Za-z0-9_-]{10,}|AKIA[0-9A-Z]{16})' \
    -- . ||
    true
)"

if [ -n "$content_hits" ]; then
  echo "Potential committed secret detected in:"
  echo "$content_hits"
  exit 1
fi

echo "Sensitive-file guard passed."
