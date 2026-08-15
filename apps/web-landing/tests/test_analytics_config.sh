#!/bin/bash
# The landing is a static export, so the OpenPanel config the Helm chart injects
# can only reach the browser through nginx's envsubst templating at boot.
# Usage: apps/web-landing/tests/test_analytics_config.sh
set -uo pipefail

APP="$(cd "$(dirname "$0")/.." && pwd)"
rendered=$(
  OPENPANEL_CLIENT_ID=test-client-id \
  OPENPANEL_API_URL=https://analytics.tesserix.app/api \
  OPENPANEL_SCRIPT_URL=https://analytics.tesserix.app/op1.js \
  envsubst '${OPENPANEL_CLIENT_ID} ${OPENPANEL_API_URL} ${OPENPANEL_SCRIPT_URL}' \
    < "$APP/nginx.conf"
)

pass=0; fail=0
check() {
  if grep -q "$2" <<<"$rendered"; then
    echo "  PASS $1"; pass=$((pass+1))
  else
    echo "  FAIL $1"; fail=$((fail+1))
  fi
}

check "the config endpoint carries the injected client id" '"clientId":"test-client-id"'
check "the config endpoint carries the api url" '"apiUrl":"https://analytics.tesserix.app/api"'
check "the config endpoint carries the script url" '"scriptUrl":"https://analytics.tesserix.app/op1.js"'
check "the tracking script is allowed by CSP" "script-src 'self' 'unsafe-inline' https://analytics.tesserix.app"
check "events may be posted to the analytics host" "connect-src 'self' https://analytics.tesserix.app"

echo
echo "passed=$pass failed=$fail"
[ "$fail" -eq 0 ]
