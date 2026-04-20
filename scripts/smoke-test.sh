#!/usr/bin/env bash

set -euo pipefail

base_url="${BASE_URL:-http://localhost:18080}"
phone="${PHONE:-+79991234567}"
size="${SIZE:-m}"
locker_id="${LOCKER_ID:-123}"
wait_seconds="${WAIT_SECONDS:-6}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}

trap cleanup EXIT

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require curl
require jq

wait_for_api() {
  local attempts="${API_WAIT_ATTEMPTS:-30}"
  local delay="${API_WAIT_DELAY:-1}"
  local i=1

  while [[ "$i" -le "$attempts" ]]; do
    if curl -sS -o /dev/null -w '%{http_code}' "$base_url/healthz" | grep -qx '200'; then
      return 0
    fi
    sleep "$delay"
    i=$((i + 1))
  done

  echo "API is not ready at $base_url/healthz" >&2
  exit 1
}

request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local output_file="$4"

  if [[ -n "$body" ]]; then
    curl -sS -o "$output_file" -w '%{http_code}' -X "$method" "$base_url$path" \
      -H 'Content-Type: application/json' \
      -d "$body"
  else
    curl -sS -o "$output_file" -w '%{http_code}' -X "$method" "$base_url$path"
  fi
}

assert_status() {
  local label="$1"
  local expected="$2"
  local actual="$3"
  local response_file="$4"

  if [[ "$actual" != "$expected" ]]; then
    echo "$label -> expected $expected, got $actual" >&2
    echo "response:" >&2
    cat "$response_file" >&2
    exit 1
  fi

  echo "$label -> $actual"
}

response_body() {
  local response_file="$1"
  jq -c '.data' "$response_file"
}

echo "base_url=$base_url"
wait_for_api

root_body="$tmp_dir/root.json"
health_body="$tmp_dir/health.json"
lockers_body="$tmp_dir/lockers.json"
selection_body="$tmp_dir/selection.json"
booking_body="$tmp_dir/booking.json"
access_body="$tmp_dir/access.json"
payment_body="$tmp_dir/payment.json"
open_before_body="$tmp_dir/open-before.json"
payment_after_body="$tmp_dir/payment-after.json"
open_after_body="$tmp_dir/open-after.json"
rental_before_body="$tmp_dir/rental-before.json"
finish_body="$tmp_dir/finish.json"
rental_after_body="$tmp_dir/rental-after.json"

code="$(request GET / '' "$root_body")"
assert_status "GET /" 200 "$code" "$root_body"

code="$(request GET /healthz '' "$health_body")"
assert_status "GET /healthz" 200 "$code" "$health_body"

code="$(request GET /api/v1/lockers '' "$lockers_body")"
assert_status "GET /api/v1/lockers" 200 "$code" "$lockers_body"

code="$(request POST "/api/v1/lockers/$locker_id/cell-selection" "{\"size\":\"$size\"}" "$selection_body")"
assert_status "POST /api/v1/lockers/$locker_id/cell-selection" 200 "$code" "$selection_body"
selection_id="$(jq -r '.data.selectionId // empty' "$selection_body")"
if [[ -z "$selection_id" ]]; then
  echo "selectionId missing in response" >&2
  cat "$selection_body" >&2
  exit 1
fi

code="$(request POST "/api/v1/lockers/$locker_id/bookings" "{\"selectionId\":\"$selection_id\",\"phone\":\"$phone\"}" "$booking_body")"
assert_status "POST /api/v1/lockers/$locker_id/bookings" 201 "$code" "$booking_body"
rental_id="$(jq -r '.data.rentalId // empty' "$booking_body")"
access_code="$(jq -r '.data.accessCode // empty' "$booking_body")"
if [[ -z "$rental_id" || -z "$access_code" ]]; then
  echo "rentalId or accessCode missing in response" >&2
  cat "$booking_body" >&2
  exit 1
fi

code="$(request POST "/api/v1/lockers/$locker_id/access-code/check" "{\"accessCode\":\"$access_code\"}" "$access_body")"
assert_status "POST /api/v1/lockers/$locker_id/access-code/check" 200 "$code" "$access_body"
payment_id="$(jq -r '.data.payment.paymentId // empty' "$access_body")"
if [[ -z "$payment_id" ]]; then
  echo "paymentId missing in response" >&2
  cat "$access_body" >&2
  exit 1
fi

code="$(request GET "/api/v1/payments/$payment_id" '' "$payment_body")"
assert_status "GET /api/v1/payments/$payment_id" 200 "$code" "$payment_body"

code="$(request POST "/api/v1/rentals/$rental_id/open" '' "$open_before_body")"
assert_status "POST /api/v1/rentals/$rental_id/open (before paid)" 402 "$code" "$open_before_body"

sleep "$wait_seconds"

code="$(request GET "/api/v1/payments/$payment_id" '' "$payment_after_body")"
assert_status "GET /api/v1/payments/$payment_id (after wait)" 200 "$code" "$payment_after_body"

code="$(request POST "/api/v1/rentals/$rental_id/open" '' "$open_after_body")"
assert_status "POST /api/v1/rentals/$rental_id/open (after paid)" 200 "$code" "$open_after_body"

code="$(request GET "/api/v1/rentals/$rental_id" '' "$rental_before_body")"
assert_status "GET /api/v1/rentals/$rental_id" 200 "$code" "$rental_before_body"

code="$(request POST "/api/v1/rentals/$rental_id/finish" '' "$finish_body")"
assert_status "POST /api/v1/rentals/$rental_id/finish" 200 "$code" "$finish_body"

code="$(request GET "/api/v1/rentals/$rental_id" '' "$rental_after_body")"
assert_status "GET /api/v1/rentals/$rental_id (after finish)" 200 "$code" "$rental_after_body"

echo "--- key payloads ---"
response_body "$access_body"
response_body "$payment_after_body"
response_body "$rental_after_body"