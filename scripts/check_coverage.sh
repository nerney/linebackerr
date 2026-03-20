#!/usr/bin/env bash
set -euo pipefail

MIN_COVERAGE="${MIN_COVERAGE:-60}"

if ! [[ "$MIN_COVERAGE" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
  echo "MIN_COVERAGE must be numeric, got: $MIN_COVERAGE" >&2
  exit 2
fi

status=0
printf "Per-package coverage (minimum: %s%%)\n" "$MIN_COVERAGE"
printf "%-35s %10s\n" "PACKAGE" "COVERAGE"

while IFS= read -r pkg; do
  if ! ls "${pkg//linebackerr\//./}"/*_test.go >/dev/null 2>&1 && [[ "$pkg" != "linebackerr" ]]; then
    echo "❌ $pkg has no *_test.go files"
    status=1
    continue
  fi

  out=$(go test -cover "$pkg" 2>&1) || {
    echo "$out"
    status=1
    continue
  }

  cov=$(printf '%s\n' "$out" | sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' | tail -n1)
  if [[ -z "$cov" ]]; then
    cov="0"
  fi

  printf "%-35s %9s%%\n" "$pkg" "$cov"

  if ! awk -v c="$cov" -v min="$MIN_COVERAGE" 'BEGIN{exit !(c+0 < min+0)}'; then
    :
  else
    echo "❌ $pkg coverage ${cov}% is below ${MIN_COVERAGE}%"
    status=1
  fi
done < <(go list ./...)

if [[ "$status" -ne 0 ]]; then
  echo "Coverage check failed"
  exit "$status"
fi

echo "Coverage check passed"
