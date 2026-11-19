#!/usr/bin/env bash
#
# SplitKita - restore dump ke PostgreSQL container di host Coolify.
#
# Jalankan di host homeserver (terminal host Coolify), bukan di dalam container.
#
# Pemakaian:
#   ./db-import-coolify.sh /tmp/splitkita-20260101-120000.dump
#   ./db-import-coolify.sh /tmp/splitkita-20260101-120000.sql
#
# Environment:
#   PG_CONTAINER  nama container PostgreSQL Coolify (wajib)
#   DB_NAME       default splitkita
#   DB_USER       default splitkita
#   DB_HOST       default 127.0.0.1
#   DB_PORT       default 5432
#   PGPASSWORD    password DB_USER (bisa juga lewat prompt psql)
#   ALLOW_NON_EMPTY=1  izinkan restore ke DB yang sudah berisi tabel
#
# Script ini TIDAK membuat database/user. Buat resource Postgres dulu di
# Coolify (DB + user sesuai env di atas), lalu restore.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DUMP="${1:-}"
DB_NAME="${DB_NAME:-splitkita}"
DB_USER="${DB_USER:-splitkita}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
PG_CONTAINER="${PG_CONTAINER:-}"
ALLOW_NON_EMPTY="${ALLOW_NON_EMPTY:-0}"

die() {
    printf 'ERROR: %s\n' "$*" >&2
    exit 1
}

[ -n "$DUMP" ] || die "Path dump belum diberikan. Contoh: ./db-import-coolify.sh /tmp/splitkita.dump"
[ -f "$DUMP" ] || die "File dump tidak ditemukan: $DUMP"
[ -n "$PG_CONTAINER" ] || die "PG_CONTAINER belum di-set. Cek: docker ps --format '{{.Names}}'"
command -v docker >/dev/null 2>&1 || die "docker tidak ditemukan di PATH."

if ! docker inspect "$PG_CONTAINER" >/dev/null 2>&1; then
    die "Container '$PG_CONTAINER' tidak ditemukan."
fi

printf '==> Container : %s\n' "$PG_CONTAINER"
printf '==> Database  : %s@%s:%s\n' "$DB_NAME" "$DB_HOST" "$DB_PORT"
printf '==> Dump      : %s (%s)\n' "$DUMP" "$(du -h "$DUMP" | cut -f1)"

psql_exec() {
    if [ -n "${PGPASSWORD:-}" ]; then
        docker exec -e PGPASSWORD="$PGPASSWORD" "$PG_CONTAINER" \
            psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" "$@"
    else
        docker exec -it "$PG_CONTAINER" \
            psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" "$@"
    fi
}

printf '==> Cek koneksi...\n'
psql_exec -Atc 'SELECT 1' >/dev/null || die "Tidak bisa connect ke database $DB_NAME sebagai $DB_USER."

existing_tables="$(psql_exec -Atc "SELECT count(*) FROM pg_tables WHERE schemaname = 'public';")"
if [ "${existing_tables:-0}" -gt 0 ] && [ "$ALLOW_NON_EMPTY" != "1" ]; then
    cat >&2 <<EOF
ERROR: database '$DB_NAME' sudah punya $existing_tables tabel.

Untuk database fresh (rekomendasi):
  - hapus resource Postgres di Coolify lalu buat ulang, ATAU
  - buat database baru yang kosong, ATAU
  - jalankan ulang dengan ALLOW_NON_EMPTY=1 (objek akan di-drop untuk dump .dump;
    untuk dump .sql restore akan gagal karena objek sudah ada).

Restore ke DB berisi data akan menimpa data lama.
EOF
    exit 1
fi

is_plain_sql=0
case "$DUMP" in
    *.sql) is_plain_sql=1 ;;
esac

container_tmp="/tmp/splitkita-restore.$([ "$is_plain_sql" -eq 1 ] && echo sql || echo dump)"

printf '==> Copy dump ke container...\n'
docker cp "$DUMP" "$PG_CONTAINER:$container_tmp"

cleanup() {
    docker exec "$PG_CONTAINER" rm -f "$container_tmp" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if [ "$is_plain_sql" -eq 1 ]; then
    printf '==> Restore plain SQL (psql, ON_ERROR_STOP=1)...\n'
    if [ -n "${PGPASSWORD:-}" ]; then
        docker exec -e PGPASSWORD="$PGPASSWORD" "$PG_CONTAINER" \
            psql -v ON_ERROR_STOP=1 -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" \
            -d "$DB_NAME" -f "$container_tmp"
    else
        docker exec -it "$PG_CONTAINER" \
            psql -v ON_ERROR_STOP=1 -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" \
            -d "$DB_NAME" -f "$container_tmp"
    fi
else
    printf '==> Restore custom dump (pg_restore)...\n'
    if [ -n "${PGPASSWORD:-}" ]; then
        docker exec -e PGPASSWORD="$PGPASSWORD" "$PG_CONTAINER" \
            pg_restore --clean --if-exists --no-owner --no-privileges --no-tablespaces \
            --exit-on-error -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" \
            -d "$DB_NAME" "$container_tmp"
    else
        docker exec -it "$PG_CONTAINER" \
            pg_restore --clean --if-exists --no-owner --no-privileges --no-tablespaces \
            --exit-on-error -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" \
            -d "$DB_NAME" "$container_tmp"
    fi
fi

printf '==> Verifikasi...\n'
if [ -f "$SCRIPT_DIR/db-verify.sql" ]; then
    if [ -n "${PGPASSWORD:-}" ]; then
        docker exec -i -e PGPASSWORD="$PGPASSWORD" "$PG_CONTAINER" \
            psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" \
            -v ON_ERROR_STOP=1 -f - < "$SCRIPT_DIR/db-verify.sql"
    else
        docker exec -i "$PG_CONTAINER" \
            psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" \
            -v ON_ERROR_STOP=1 -f - < "$SCRIPT_DIR/db-verify.sql"
    fi
else
    printf 'WARN: db-verify.sql tidak ditemukan di %s, verifikasi dilewati.\n' "$SCRIPT_DIR" >&2
fi

printf '\nSelesai. Bandingkan row count dengan hasil db-export lokal.\n'
printf 'Langkah berikutnya: set env backend di Coolify lalu deploy.\n'
