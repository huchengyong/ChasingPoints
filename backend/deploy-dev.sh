#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
    printf 'Usage: CHASING_POINTS_DEPLOY_TARGET=user@host %s\n' "${0##*/}"
    printf '       %s user@host\n' "${0##*/}"
}

case "${1:-}" in
    -h|--help)
        usage
        exit 0
        ;;
esac

deploy_target="${CHASING_POINTS_DEPLOY_TARGET:-${1:-}}"
deploy_port="${CHASING_POINTS_DEPLOY_PORT:-22}"
if [ -z "$deploy_target" ]; then
    usage >&2
    exit 2
fi

backend_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
revision="$(git -C "$backend_dir" rev-parse --short=12 HEAD)"
if [ -n "$(git -C "$backend_dir" status --porcelain -- .)" ]; then
    revision="${revision}-dirty"
fi
release_id="$(date +%Y%m%d%H%M%S)-${revision}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/chasing-points-deploy.XXXXXX")"
archive_name="chasing-points-${release_id}.tar.gz"
archive="$work_dir/$archive_name"
remote_archive="/tmp/$archive_name"
trap 'rm -rf "$work_dir"' EXIT

printf 'Testing backend...\n'
(cd "$backend_dir" && go test ./...)

printf 'Building Linux ARM64 release %s...\n' "$release_id"
(cd "$backend_dir" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o "$work_dir/chasing-points" .)
cp "$backend_dir/etc/chasing_points-api.yaml" "$work_dir/chasing_points-api.yaml"
cp -R "$backend_dir/migrations" "$work_dir/migrations"
tar -C "$work_dir" -czf "$archive" chasing-points chasing_points-api.yaml migrations

printf 'Uploading release...\n'
scp -P "$deploy_port" "$archive" "${deploy_target}:${remote_archive}"

printf 'Activating release...\n'
ssh -p "$deploy_port" "$deploy_target" "sudo bash -s -- '$release_id' '$remote_archive'" <<'REMOTE'
set -Eeuo pipefail

release_id=$1
uploaded_archive=$2
base=/www/server/chasing-points
release="$base/releases/$release_id"
previous="$(readlink -f "$base/current" || true)"
created=0
switched=0

finish() {
    status=$?
    trap - EXIT
    rm -f "$uploaded_archive"
    if [ "$status" -ne 0 ]; then
        if [ "$switched" -eq 1 ] && [ -n "$previous" ]; then
            ln -sfn "$previous" "$base/current"
            systemctl restart chasing-points.service || true
        fi
        if [ "$created" -eq 1 ]; then
            rm -rf "$release"
        fi
    fi
    exit "$status"
}
trap finish EXIT

[ -n "$previous" ] || { printf 'Current release not found\n' >&2; exit 1; }
[ -x "$base/current/goose" ] || { printf 'Goose binary not found\n' >&2; exit 1; }
[ -f "$base/shared/.env" ] || { printf 'Shared environment file not found\n' >&2; exit 1; }
[ ! -e "$release" ] || { printf 'Release already exists: %s\n' "$release" >&2; exit 1; }

mkdir -p "$release"
created=1
tar -xzf "$uploaded_archive" -C "$release"
cp "$base/current/goose" "$release/goose"
chmod 755 "$release/chasing-points" "$release/goose"
chown -R www:www "$release"

mysql_dsn="$(sed -n '/^MYSQL_DSN=/{s/^MYSQL_DSN=//;p;q;}' "$base/shared/.env")"
[ -n "$mysql_dsn" ] || { printf 'MYSQL_DSN is missing from shared environment\n' >&2; exit 1; }
"$release/goose" -dir "$release/migrations" mysql "$mysql_dsn" up

ln -sfn "$release" "$base/current"
switched=1
systemctl restart chasing-points.service
curl --fail --silent --show-error --retry 10 --retry-delay 1 --retry-connrefused \
    http://127.0.0.1:8878/api/admin/exists >/dev/null

printf 'Deployed %s\n' "$release_id"
REMOTE

printf 'Deployment complete: https://dev-api.kekemate.cn\n'
