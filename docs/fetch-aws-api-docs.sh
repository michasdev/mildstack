#!/usr/bin/env bash
#
# fetch-aws-api-docs.sh
#
# Downloads AWS API Reference documentation (markdown) for a given service
# and organizes it into a structured folder under docs/aws/<service>/
#
# Usage:
#   ./docs/fetch-aws-api-docs.sh <base_url> <service_alias>
#
# Examples:
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/sns/latest/api sns
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/AmazonS3/latest/API s3
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/amazondynamodb/latest/APIReference dynamodb
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference sqs
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/lambda/latest/api lambda
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/eventbridge/latest/APIReference eventbridge
#   ./docs/fetch-aws-api-docs.sh https://docs.aws.amazon.com/systems-manager/latest/APIReference ssm
#
# The base_url is the path up to (but not including) the .md filenames.
# The service_alias is the short name used for the output folder.
#
# Output structure:
#   docs/aws/<service>/actions/<ActionName>.md
#   docs/aws/<service>/datatypes/<TypeName>.md
#   docs/aws/<service>/common-parameters.md
#   docs/aws/<service>/common-errors.md
#

set -euo pipefail

# ── Colors ──────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

# ── Config ──────────────────────────────────────────────────────────
MAX_CONCURRENT=8          # parallel download limit
RETRY_COUNT=3             # retries per file
RETRY_DELAY=2             # seconds between retries
REQUEST_DELAY=0.1         # seconds between launching downloads (rate limit)

# ── Helpers ─────────────────────────────────────────────────────────

log_info()    { echo -e "${CYAN}ℹ${RESET}  $*"; }
log_success() { echo -e "${GREEN}✔${RESET}  $*"; }
log_warn()    { echo -e "${YELLOW}⚠${RESET}  $*"; }
log_error()   { echo -e "${RED}✖${RESET}  $*"; }
log_step()    { echo -e "\n${BOLD}${CYAN}── $* ──${RESET}"; }

usage() {
  echo -e "${BOLD}Usage:${RESET} $0 <base_url> <service_alias>"
  echo ""
  echo "  base_url        AWS API docs base URL (without trailing slash)"
  echo "  service_alias    Short name for output folder (e.g. sns, s3, dynamodb)"
  echo ""
  echo -e "${DIM}Examples:${RESET}"
  echo "  $0 https://docs.aws.amazon.com/sns/latest/api sns"
  echo "  $0 https://docs.aws.amazon.com/AmazonS3/latest/API s3"
  echo "  $0 https://docs.aws.amazon.com/amazondynamodb/latest/APIReference dynamodb"
  exit 1
}

# Download a single .md file with retries
# Args: $1=url $2=output_path
download_md() {
  local url="$1"
  local output="$2"
  local attempt=0

  while (( attempt < RETRY_COUNT )); do
    attempt=$((attempt + 1))
    local http_code
    http_code=$(curl -sS -w "%{http_code}" -o "$output" "$url" 2>/dev/null) || true

    if [[ "$http_code" == "200" ]]; then
      return 0
    fi

    if [[ "$http_code" == "404" ]]; then
      rm -f "$output"
      return 1
    fi

    if (( attempt < RETRY_COUNT )); then
      sleep "$RETRY_DELAY"
    fi
  done

  rm -f "$output"
  return 1
}

# Parse an index markdown and extract all linked .md filenames
# Args: $1=path_to_index_md
# Output: one filename per line (e.g. "API_Publish.md")
extract_linked_files() {
  local index_file="$1"
  # Match patterns like (API_Something.md) or (API_control_Something.md)
  grep -oE '\(API_[^)]+\.md\)' "$index_file" 2>/dev/null \
    | sed 's/^(//; s/)$//' \
    | sort -u
}

# Download files in parallel with a progress counter
# Args: $1=base_url $2=output_dir $3...=filenames
download_batch() {
  local base_url="$1"
  local output_dir="$2"
  shift 2
  local files=("$@")
  local total=${#files[@]}
  local completed=0
  local failed=0
  local pids=()
  local pid_to_file=()

  for file in "${files[@]}"; do
    # Derive a clean name: strip API_ prefix and .md suffix
    local clean_name
    clean_name=$(echo "$file" | sed 's/^API_//; s/\.md$//')
    local output_path="${output_dir}/${clean_name}.md"
    local url="${base_url}/${file}"

    # Launch download in background
    (download_md "$url" "$output_path") &
    local pid=$!
    pids+=("$pid")
    pid_to_file+=("$pid:$file")

    # Rate limit launches
    sleep "$REQUEST_DELAY"

    # Throttle: wait if we hit max concurrent
    while (( $(jobs -rp | wc -l) >= MAX_CONCURRENT )); do
      sleep 0.2
    done
  done

  # Wait for all downloads and collect results
  for i in "${!pids[@]}"; do
    local pid="${pids[$i]}"
    local entry="${pid_to_file[$i]}"
    local file="${entry#*:}"

    if wait "$pid" 2>/dev/null; then
      completed=$((completed + 1))
    else
      failed=$((failed + 1))
      log_warn "Failed to download: ${file}"
    fi

    # Progress
    local done_count=$((completed + failed))
    printf "\r  ${DIM}[%d/%d]${RESET} downloaded (%d failed)" "$done_count" "$total" "$failed"
  done
  echo ""  # newline after progress

  if (( failed > 0 )); then
    log_warn "${failed}/${total} files failed to download"
  else
    log_success "All ${total} files downloaded successfully"
  fi
}

# ── Main ────────────────────────────────────────────────────────────

if [[ $# -lt 2 ]]; then
  usage
fi

BASE_URL="${1%/}"   # strip trailing slash
SERVICE="$2"

# Resolve project root (relative to this script)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/docs/aws/${SERVICE}"

echo ""
echo -e "${BOLD}AWS API Docs Fetcher${RESET}"
echo -e "${DIM}Service: ${SERVICE} | Base: ${BASE_URL}${RESET}"
echo ""

# Create output dirs
mkdir -p "${OUTPUT_DIR}/actions"
mkdir -p "${OUTPUT_DIR}/datatypes"

# ── 1. Fetch & parse API_Operations.md ──────────────────────────────
log_step "Actions (API_Operations)"

OPERATIONS_INDEX=$(mktemp)
trap "rm -f '$OPERATIONS_INDEX'" EXIT

if download_md "${BASE_URL}/API_Operations.md" "$OPERATIONS_INDEX"; then
  # Save the index itself
  cp "$OPERATIONS_INDEX" "${OUTPUT_DIR}/actions/_index.md"

  # Extract linked files
  ACTION_FILES=()
  while IFS= read -r line; do
    ACTION_FILES+=("$line")
  done < <(extract_linked_files "$OPERATIONS_INDEX")
  ACTION_COUNT=${#ACTION_FILES[@]}

  if (( ACTION_COUNT == 0 )); then
    log_warn "No actions found in API_Operations.md"
  else
    log_info "Found ${BOLD}${ACTION_COUNT}${RESET} actions"
    download_batch "$BASE_URL" "${OUTPUT_DIR}/actions" "${ACTION_FILES[@]}"
  fi
else
  log_warn "API_Operations.md not found — skipping actions"
fi

# ── 2. Fetch & parse API_Types.md ──────────────────────────────────
log_step "Data Types (API_Types)"

TYPES_INDEX=$(mktemp)
trap "rm -f '$OPERATIONS_INDEX' '$TYPES_INDEX'" EXIT

if download_md "${BASE_URL}/API_Types.md" "$TYPES_INDEX"; then
  # Save the index itself
  cp "$TYPES_INDEX" "${OUTPUT_DIR}/datatypes/_index.md"

  # Extract linked files
  TYPE_FILES=()
  while IFS= read -r line; do
    TYPE_FILES+=("$line")
  done < <(extract_linked_files "$TYPES_INDEX")
  TYPE_COUNT=${#TYPE_FILES[@]}

  if (( TYPE_COUNT == 0 )); then
    log_warn "No data types found in API_Types.md"
  else
    log_info "Found ${BOLD}${TYPE_COUNT}${RESET} data types"
    download_batch "$BASE_URL" "${OUTPUT_DIR}/datatypes" "${TYPE_FILES[@]}"
  fi
else
  log_warn "API_Types.md not found — skipping data types"
fi

# ── 3. CommonParameters.md ─────────────────────────────────────────
log_step "Common Parameters"

if download_md "${BASE_URL}/CommonParameters.md" "${OUTPUT_DIR}/common-parameters.md"; then
  log_success "Downloaded common-parameters.md"
else
  log_warn "CommonParameters.md not found — skipping"
fi

# ── 4. CommonErrors.md ─────────────────────────────────────────────
log_step "Common Errors"

if download_md "${BASE_URL}/CommonErrors.md" "${OUTPUT_DIR}/common-errors.md"; then
  log_success "Downloaded common-errors.md"
else
  log_warn "CommonErrors.md not found — skipping"
fi

# ── Summary ─────────────────────────────────────────────────────────
echo ""
log_step "Summary"

TOTAL_FILES=$(find "${OUTPUT_DIR}" -name "*.md" | wc -l | tr -d ' ')
TOTAL_SIZE=$(du -sh "${OUTPUT_DIR}" 2>/dev/null | cut -f1)

log_success "Downloaded ${BOLD}${TOTAL_FILES}${RESET} files (${TOTAL_SIZE}) → ${DIM}${OUTPUT_DIR}${RESET}"

echo ""
echo -e "${DIM}Directory structure:${RESET}"
echo -e "  docs/aws/${SERVICE}/"
echo -e "  ├── actions/          (${ACTION_COUNT:-0} operations)"
echo -e "  ├── datatypes/        (${TYPE_COUNT:-0} types)"
echo -e "  ├── common-parameters.md"
echo -e "  └── common-errors.md"
echo ""
