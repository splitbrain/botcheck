#!/usr/bin/env bash
set -euo pipefail

# Usage: ./update-bot-addresses.sh
# Fetches official bot IP JSON/text documents and refreshes apache/addresses.net.list.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIST_FILE="${LIST_FILE:-${SCRIPT_DIR}/apache/addresses.net.list}"

SOURCES=(
  "Googlebot|https://developers.google.com/search/apis/ipranges/googlebot.json"
  "Bingbot|https://www.bing.com/toolbox/bingbot.json"
  "GPT User Bot|https://openai.com/chatgpt-user.json"
  "Yandexbot|https://raw.githubusercontent.com/sefinek/known-bots-ip-whitelist/main/lists/yandexbot/ips.txt"
)

SECTION_START_PREFIX="# BOTCHECK_SOURCE_START:"
SECTION_END_PREFIX="# BOTCHECK_SOURCE_END:"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

tmp_out="$(mktemp)"
tmp_files=("$tmp_out")
cleanup() {
  rm -f "${tmp_files[@]}"
}
trap cleanup EXIT

header_line="# Add IP addresses or CIDR blocks here, one per line."
if [[ -s "$LIST_FILE" ]]; then
  first_line="$(head -n 1 "$LIST_FILE")"
  if [[ "$first_line" == \#* ]]; then
    header_line="$first_line"
  fi
fi

declare -A block_content
declare -A block_processed
block_order=()

for entry in "${SOURCES[@]}"; do
  IFS='|' read -r name url <<<"$entry"
  tmp_body="$(mktemp)"
  tmp_hdr="$(mktemp)"
  tmp_files+=("$tmp_body" "$tmp_hdr")

  curl -fsSL -D "$tmp_hdr" -o "$tmp_body" "$url"

  content_type="$(awk 'BEGIN{IGNORECASE=1} /^content-type:/ {gsub(/\r$/,""); sub(/^content-type:[[:space:]]*/i,""); sub(/;.*/,""); print; exit}' "$tmp_hdr")"
  is_json=0
  if [[ "$content_type" =~ json ]]; then
    is_json=1
  fi

  block="${SECTION_START_PREFIX} $name $url"$'\n'
  if [[ "$is_json" -eq 1 ]]; then
    mapfile -t ranges < <(
      jq -r '.. | objects | [.ipv4Prefix?, .ipv6Prefix?] | .[]? | select(. != null and . != "")' "$tmp_body" |
        sort -u
    )
  else
    mapfile -t ranges < <(
      sed 's/[[:space:]]*$//' "$tmp_body" |
        grep -Ev '^\s*(#|$)' |
        sort -u
    )
  fi
  if [[ "${#ranges[@]}" -eq 0 ]]; then
    block+="# No ranges found from $url"$'\n'
  else
    for r in "${ranges[@]}"; do
      block+="$r"$'\n'
    done
  fi
  block+="${SECTION_END_PREFIX} $name"$'\n\n'
  block_content["$name"]="$block"
  block_order+=("$name")
done

if [[ -f "$LIST_FILE" ]]; then
  skip_blank_after_block=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    trimmed="${line#"${line%%[![:space:]]*}"}"
    if [[ "$skip_blank_after_block" -eq 1 && "$trimmed" =~ ^[[:space:]]*$ ]]; then
      continue
    fi
    skip_blank_after_block=0
    if [[ "$trimmed" =~ ^#\ BOTCHECK_SOURCE_START: ]]; then
      rest="${trimmed#${SECTION_START_PREFIX} }"
      name="${rest% *}"
      if [[ -n "${block_content[$name]:-}" ]]; then
        printf "%s" "${block_content[$name]}" >>"$tmp_out"
        block_processed["$name"]=1
      else
        echo "$line" >>"$tmp_out"
      fi
      while IFS= read -r inner || [[ -n "$inner" ]]; do
        inner_trim="${inner#"${inner%%[![:space:]]*}"}"
        if [[ "$inner_trim" =~ ^#\ BOTCHECK_SOURCE_END: ]]; then
          if [[ -z "${block_content[$name]:-}" ]]; then
            echo "$inner" >>"$tmp_out"
          fi
          break
        fi
        if [[ -z "${block_content[$name]:-}" ]]; then
          echo "$inner" >>"$tmp_out"
        fi
      done
      skip_blank_after_block=1
      continue
    fi
    echo "$line" >>"$tmp_out"
  done <"$LIST_FILE"
else
  echo "$header_line" >>"$tmp_out"
  echo >>"$tmp_out"
fi

for name in "${block_order[@]}"; do
  if [[ -n "${block_processed[$name]:-}" ]]; then
    continue
  fi
  if [[ -s "$tmp_out" && -n "$(tail -c 1 "$tmp_out")" ]]; then
    echo >>"$tmp_out"
  fi
  printf "%s" "${block_content[$name]}" >>"$tmp_out"
done

mv "$tmp_out" "$LIST_FILE"
