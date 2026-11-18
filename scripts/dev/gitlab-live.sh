#!/usr/bin/env bash
# Opt-in GitLab CE live verification helpers.
# Usage: ./scripts/dev/gitlab-live.sh <up|wait-ready|seed|down|smoke|help> [--volumes]
#        ./scripts/dev/gitlab-live.sh --help

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../dist/shared/_lib.sh
source "${SCRIPT_DIR}/../dist/shared/_lib.sh"

COMPOSE_FILE="${REPO_ROOT}/docker/gitlab/docker-compose.yml"
ENV_FILE="${REPO_ROOT}/docker/gitlab/.env"
TOKEN_DIR="${REPO_ROOT}/tmp/gitlab-live"
TOKEN_FILE="${TOKEN_DIR}/token"
DEFAULT_URL="http://localhost:8929"
DEFAULT_PROJECT="root/tokotachi-live"
CONTAINER_NAME="tokotachi-gitlab"
WAIT_TIMEOUT_SEC="${GITLAB_WAIT_TIMEOUT_SEC:-900}"
GLAB_NI="${REPO_ROOT}/scripts/dev/gitlab-glab-ni.sh"

usage() {
  cat <<EOF
Usage: ./scripts/dev/gitlab-live.sh <command> [options]

Commands:
  up            Start GitLab CE (docker compose up -d)
  wait-ready    Wait until HTTP readiness succeeds (timeout ${WAIT_TIMEOUT_SEC}s)
  seed          Create project + PAT; write token to tmp/gitlab-live/token
  down          Stop containers (add --volumes to wipe named volumes)
  smoke         Require TT_LIVE_GITLAB=1; build + live-gitlab integration tests
  help          Show this help

Options:
  --help        Show this help
  --volumes     With down: also remove named volumes
  --rotate-token  With seed: always mint a new PAT

Environment:
  TT_LIVE_GITLAB=1          Required for smoke
  GITLAB_HTTP_URL           Default ${DEFAULT_URL}
  GITLAB_PROJECT_PATH       Default ${DEFAULT_PROJECT}
  GITLAB_ROOT_PASSWORD       Optional; used by seed if set / in .env
  GITLAB_WAIT_TIMEOUT_SEC   wait-ready timeout (default 900)

See docker/gitlab/README.md for prerequisites (RAM, ports, auth).
EOF
}

load_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    # shellcheck disable=SC1090
    set -a
    source "$ENV_FILE"
    set +a
  fi
}

compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
  else
    fail "docker compose / docker-compose not found"
    exit 1
  fi
}

http_url() {
  echo "${GITLAB_HTTP_URL:-$DEFAULT_URL}"
}

project_path() {
  echo "${GITLAB_PROJECT_PATH:-$DEFAULT_PROJECT}"
}

cmd_up() {
  local dc
  dc="$(compose_cmd)"
  info "Starting GitLab CE via ${COMPOSE_FILE}"
  # Optional root password from .env on first boot
  if [[ -f "$ENV_FILE" ]] && grep -q '^GITLAB_ROOT_PASSWORD=.' "$ENV_FILE" 2>/dev/null; then
    # shellcheck disable=SC1090
    set -a
    source "$ENV_FILE"
    set +a
    if [[ -n "${GITLAB_ROOT_PASSWORD:-}" ]]; then
      export GITLAB_ROOT_PASSWORD
    fi
  fi
  $dc -f "$COMPOSE_FILE" up -d
  pass "GitLab container started (first boot may take 5–15 minutes)"
  info "Next: ./scripts/dev/gitlab-live.sh wait-ready"
}

cmd_wait_ready() {
  local url ready_url start now elapsed
  url="$(http_url)"
  ready_url="${url}/-/readiness"
  start="$(date +%s)"
  info "Waiting for readiness at ${ready_url} (timeout ${WAIT_TIMEOUT_SEC}s)"
  while true; do
    if curl -fsS --max-time 5 "$ready_url" >/dev/null 2>&1; then
      now="$(date +%s)"
      elapsed=$((now - start))
      pass "GitLab ready after ${elapsed}s"
      return 0
    fi
    now="$(date +%s)"
    elapsed=$((now - start))
    if (( elapsed >= WAIT_TIMEOUT_SEC )); then
      fail "Timed out after ${elapsed}s waiting for ${ready_url}"
      exit 1
    fi
    sleep 5
  done
}

read_root_password() {
  if [[ -n "${GITLAB_ROOT_PASSWORD:-}" ]]; then
    echo "$GITLAB_ROOT_PASSWORD"
    return 0
  fi
  if docker exec "$CONTAINER_NAME" test -f /etc/gitlab/initial_root_password 2>/dev/null; then
    docker exec "$CONTAINER_NAME" bash -lc "grep -E '^Password:' /etc/gitlab/initial_root_password | awk '{print \$2}'" | tr -d '\r'
    return 0
  fi
  fail "Root password unknown. Set GITLAB_ROOT_PASSWORD in docker/gitlab/.env or ensure initial_root_password exists in the container."
  exit 1
}

cmd_seed() {
  local rotate=false
  for a in "$@"; do
    [[ "$a" == "--rotate-token" ]] && rotate=true
  done

  cmd_wait_ready

  local url proj token
  url="$(http_url)"
  proj="$(project_path)"
  mkdir -p "$TOKEN_DIR"

  if [[ -f "$TOKEN_FILE" && "$rotate" != "true" ]]; then
    token="$(tr -d '\r\n' <"$TOKEN_FILE")"
    if [[ -n "$token" ]] && curl -fsS --max-time 10 -H "PRIVATE-TOKEN: ${token}" "${url}/api/v4/user" >/dev/null 2>&1; then
      info "Reusing existing token at ${TOKEN_FILE}"
      ensure_project "$url" "$token" "$proj"
      print_seed_summary "$url" "$proj"
      return 0
    fi
    warn "Existing token invalid; minting a new one"
  fi

  info "Creating PAT via gitlab-rails runner"
  token="$(docker exec "$CONTAINER_NAME" gitlab-rails runner "
u = User.find_by_username('root') || User.find(1)
raise 'root user not found' unless u
t = u.personal_access_tokens.create!(
  name: 'tokotachi-live-' + Time.now.to_i.to_s,
  scopes: ['api', 'read_api', 'read_repository', 'write_repository'],
  expires_at: 365.days.from_now
)
puts t.token
" | tr -d '\r' | tail -n 1)"

  if [[ -z "$token" || "$token" == *"Error"* ]]; then
    fail "Failed to create PAT via rails runner"
    exit 1
  fi

  umask 077
  printf '%s\n' "$token" >"$TOKEN_FILE"
  chmod 600 "$TOKEN_FILE" 2>/dev/null || true

  ensure_project "$url" "$token" "$proj"
  print_seed_summary "$url" "$proj"
}

ensure_project() {
  local url="$1" token="$2" proj="$3"
  local enc path_resp code name
  name="${proj##*/}"
  enc="$(python -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$proj")"
  path_resp="$(curl -sS -o /tmp/tokotachi-gitlab-project.json -w '%{http_code}' \
    -H "PRIVATE-TOKEN: ${token}" "${url}/api/v4/projects/${enc}" || true)"
  if [[ "$path_resp" == "200" ]]; then
    info "Project already exists: ${proj}"
    return 0
  fi
  info "Creating project ${proj}"
  code="$(curl -sS -o /tmp/tokotachi-gitlab-project-create.json -w '%{http_code}' \
    -X POST -H "PRIVATE-TOKEN: ${token}" \
    --data-urlencode "name=${name}" \
    --data-urlencode "path=${name}" \
    --data "visibility=private" \
    --data "initialize_with_readme=true" \
    "${url}/api/v4/projects")"
  if [[ "$code" != "201" && "$code" != "200" ]]; then
    fail "Project create failed HTTP ${code}: $(cat /tmp/tokotachi-gitlab-project-create.json 2>/dev/null || true)"
    exit 1
  fi
  pass "Project created: ${proj}"
}

print_seed_summary() {
  local url="$1" proj="$2"
  pass "Seed complete"
  echo "  URL:          ${url}"
  echo "  Project:      ${proj}"
  echo "  Token file:   ${TOKEN_FILE}"
  echo "  (token value not printed)"
}

cmd_down() {
  local dc volumes=false
  dc="$(compose_cmd)"
  for a in "$@"; do
    [[ "$a" == "--volumes" ]] && volumes=true
  done
  if [[ "$volumes" == "true" ]]; then
    info "Stopping GitLab and removing volumes"
    $dc -f "$COMPOSE_FILE" down -v
  else
    info "Stopping GitLab (volumes preserved)"
    $dc -f "$COMPOSE_FILE" down
  fi
  pass "GitLab stopped"
}

cmd_smoke() {
  if [[ "${TT_LIVE_GITLAB:-}" != "1" ]]; then
    fail "TT_LIVE_GITLAB=1 is required to run smoke (refusing silent success)"
    echo "Example: TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke" >&2
    exit 1
  fi
  if [[ ! -f "$TOKEN_FILE" ]]; then
    fail "Token file missing: ${TOKEN_FILE}. Run: ./scripts/dev/gitlab-live.sh seed"
    exit 1
  fi
  if ! command -v glab >/dev/null 2>&1 && [[ -z "${TT_CMD_GLAB_REAL:-}" ]]; then
    fail "glab not found on PATH (or set TT_CMD_GLAB_REAL)"
    exit 1
  fi

  cmd_wait_ready

  export GITLAB_TOKEN
  GITLAB_TOKEN="$(tr -d '\r\n' <"$TOKEN_FILE")"
  export GITLAB_HOST
  GITLAB_HOST="$(http_url)"
  export TT_FORGE=gitlab
  if [[ "$(detect_os)" == "windows" ]]; then
    export TT_CMD_GLAB="${REPO_ROOT}/scripts/dev/gitlab-glab-ni.cmd"
  else
    export TT_CMD_GLAB="$GLAB_NI"
  fi
  export TT_LIVE_GITLAB=1

  info "Building then running live-gitlab tests"
  "${REPO_ROOT}/scripts/process/build.sh"
  "${REPO_ROOT}/scripts/process/integration_test.sh" --categories live-gitlab --specify "LiveGitLab"
}

main() {
  load_env_file

  if [[ $# -lt 1 ]]; then
    usage
    exit 1
  fi

  local cmd="$1"
  shift || true

  case "$cmd" in
    -h|--help|help)
      usage
      ;;
    up)
      cmd_up "$@"
      ;;
    wait-ready)
      cmd_wait_ready "$@"
      ;;
    seed)
      cmd_seed "$@"
      ;;
    down)
      cmd_down "$@"
      ;;
    smoke)
      cmd_smoke "$@"
      ;;
    *)
      fail "Unknown command: ${cmd}"
      usage
      exit 1
      ;;
  esac
}

main "$@"
