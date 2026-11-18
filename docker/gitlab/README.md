# GitLab CE live verification (opt-in)

Optional local GitLab for smoking `tt doctor` / `tt list --update` / `tt pr` against real `glab`.

This does **not** replace the default acceptance path (unit Recorder + `tt` doctor host tests). Default `integration_test.sh` / category `tt` never starts this stack.

## Prerequisites

- Docker Desktop (or equivalent Docker Engine + Compose)
- RAM **8GB+** free for GitLab CE (8–16GB recommended)
- Several GB of disk for image + volumes
- First boot typically **5–15 minutes**
- `glab` on `PATH` for smoke tests
- Bash (Git Bash on Windows)

## Ports

| Service | Host |
|---|---|
| HTTP | `8929` |
| SSH (git) | `2224` |
| External URL | `http://localhost:8929` |

Pinned image: `gitlab/gitlab-ce:17.7.0-ce.0` (see `docker-compose.yml`).

## Commands

```bash
./scripts/dev/gitlab-live.sh up
./scripts/dev/gitlab-live.sh wait-ready
./scripts/dev/gitlab-live.sh seed
# Ensure glab can reach the instance (token file is enough for smoke env):
#   export GITLAB_TOKEN="$(cat tmp/gitlab-live/token)"
#   export GITLAB_HOST=http://localhost:8929
TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke
./scripts/dev/gitlab-live.sh down
./scripts/dev/gitlab-live.sh down --volumes   # wipe named volumes
```

`smoke` requires `TT_LIVE_GITLAB=1`. Without it the script exits non-zero (does not silently succeed).

## Auth

Seed writes a root PAT to `tmp/gitlab-live/token` (under gitignored `tmp/`).

Smoke exports:

- `GITLAB_TOKEN` from that file
- `GITLAB_HOST=localhost:8929` (no scheme; HTTPS is the glab default)
- `GLAB_CONFIG_DIR` with `api_protocol: http` for this host
- `TT_FORGE=gitlab`
- `TT_CMD_GLAB` → `scripts/dev/gitlab-glab-ni.sh` / `.cmd` (adds `--yes --fill` to `glab mr create`)

Optional interactive login:

```bash
glab auth login --hostname localhost:8929 --api-host localhost:8929 --api-protocol http --token "$(cat tmp/gitlab-live/token)"
```

## What to store / not store

| Store in git | Do not store |
|---|---|
| Compose (pinned image) | Personal access tokens / root password real values |
| `gitlab-live.sh` / `gitlab-glab-ni.sh` | `docker/gitlab/.env` |
| This README / `.env.example` | GitLab data volumes (Docker-managed) |
| Opt-in tests under `tests/live-gitlab/` | Default CI job that boots GitLab CE |

## Root password (first boot)

If `GITLAB_ROOT_PASSWORD` was not set in `.env` before first boot:

```bash
docker exec tokotachi-gitlab grep 'Password:' /etc/gitlab/initial_root_password
```

The file is removed ~24h after first start on some versions; re-seed or set password via rails if needed.

## Relation to GitLab MR support (001)

| Layer | Role |
|---|---|
| Always-on | `internal/forge` unit tests + `tests/tt` doctor host tests |
| Opt-in (this) | Real CE + `glab` confidence |

## Troubleshooting

- **Port in use**: change host ports in `docker-compose.yml` and update this README / seed URL.
- **OOM / slow boot**: give Docker more RAM; re-run `wait-ready` (default timeout 900s).
- **`glab` SSL/HTTP errors**: use `GITLAB_HOST=http://localhost:8929` (HTTP, not HTTPS).
- **Smoke without prep**: expect failure (`TT_LIVE_GITLAB` / token / readiness) — `t.Skip` is not used.
