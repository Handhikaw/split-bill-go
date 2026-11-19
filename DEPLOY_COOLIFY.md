# Deploy SplitKita ke Coolify + migrasi database

Panduan ini memindahkan database dari PostgreSQL lokal (Docker Desktop) ke
PostgreSQL 18 di Coolify, lalu menjalankan backend Go di Coolify juga.

```
Browser
   │ HTTPS
   ▼
Frontend (Coolify / Vercel / dll)
   │ HTTPS + X-API-Key
   ▼
Backend Go (Coolify) ──internal network──▶ PostgreSQL 18 (Coolify)
                                              tanpa public port
```

Prinsip:

- Database **tidak** diekspos ke internet.
- Backend connect lewat network internal Coolify.
- Local dev tetap memakai Postgres lokal, bukan DB production.
- Dump lokal dibuat di laptop, transfer ke homeserver hanya saat restore.

---

## 1. Export database lokal

Stop backend lokal dulu supaya tidak ada write saat dump.

```powershell
cd C:\Users\handh\OneDrive\Documents\split-bill-go
.\scripts\db-export.ps1
```

Output di `backups/`:

| File | Fungsi |
| --- | --- |
| `splitkita-<ts>.sql` | plain SQL portable, untuk initialization script Coolify |
| `splitkita-<ts>.dump` | custom format, untuk `pg_restore` (fallback) |
| `splitkita-<ts>-verify.txt` | row count + cek objek migration |
| `splitkita-<ts>.sha256` | checksum |
| `splitkita-<ts>-manifest.txt` | ringkasan |

Kalau database lokal butuh password:

```powershell
$env:PGPASSWORD = "password-lokal"
.\scripts\db-export.ps1
```

Catatan versi: dump dari PostgreSQL **18.6**. Restore hanya ke PostgreSQL
**18+**. Jangan ke 16/17. Dump plain SQL memakai `\restrict` (psql 18), jadi
init script harus dijalankan oleh `psql` 18 — image Postgres 18 Coolify sudah
memenuhinya.

---

## 2. Buat PostgreSQL di Coolify

1. Project → **+ New Resource** → **Database** → **PostgreSQL**.
2. Pilih image **PostgreSQL 18**.
3. Isi:
   - Name: `splitkita-db`
   - Database: `splitkita`
   - User: `splitkita`
   - Password: generate password kuat, simpan di password manager.
4. **Public port: matikan** (biarkan internal network saja).
5. Tab **Initialization scripts** → tambah `001_restore.sql` → paste seluruh
   isi `backups/splitkita-<ts>.sql`.
6. Deploy / Start resource.

Init script hanya jalan **sekali**, saat volume Postgres pertama kali dibuat.
Kalau resource sudah pernah start dengan volume yang sama, script baru tidak
dijalankan.

### Kalau init script gagal

Hapus resource (volume masih fresh), buat ulang, lalu ulangi. Kalau masih
gagal, pakai fallback restore dari host:

1. Upload dump ke homeserver, mis. `/tmp/splitkita-<ts>.dump`
   (scp kalau sejaringan, atau `curl` dari temporary URL).
2. Di terminal host Coolify:

```bash
cd /root   # atau folder tempat script ini disalin
export PG_CONTAINER="$(docker ps --format '{{.Names}}' | grep -i splitkita | head -n1)"
export DB_NAME=splitkita
export DB_USER=splitkita
export PGPASSWORD='password-db'
./db-import-coolify.sh /tmp/splitkita-<ts>.dump
```

Script `db-import-coolify.sh` dan `db-verify.sql` ada di folder `scripts/`.
Copy keduanya ke host lebih dulu, lalu `chmod +x db-import-coolify.sh`.

### Verifikasi

Lewat terminal host:

```bash
docker exec -it "$PG_CONTAINER" psql -U splitkita -d splitkita -f - < db-verify.sql
```

Atau dari Coolify terminal database:

```sql
\dt
SELECT count(*) FROM bill;
```

Bandingkan angka dengan `backups/splitkita-<ts>-verify.txt`. Jumlah tabel harus
**14**.

---

## 3. Deploy backend di Coolify

1. **+ New Resource** → **Application** → repository `split-bill-go`.
2. Build pack: **Dockerfile** (sudah ada di root repo).
3. Port: `8080`.
4. Health check path: `/health`.
5. Domain: mis. `https://api.splitkita.id` (HTTPS otomatis Coolify).
6. Environment:

```env
APP_NAME=split-bill-go
APP_PORT=8080
APP_ENV=production
JWT_SECRET=<sama dengan .env lama>
GOOGLE_CLIENT_ID=<client id Google>

DB_HOST=<internal host Postgres Coolify>
DB_PORT=5432
DB_USER=splitkita
DB_PASSWORD=<password DB>
DB_NAME=splitkita

# Origin frontend persis, dipisah koma. Kosongkan kalau same-origin.
CORS_ALLOWED_ORIGINS=https://splitkita.id
```

Penting:

- `DB_HOST` **bukan** `localhost`. Ambil dari URL internal Postgres di Coolify
  (biasanya nama service/container atau UUID resource).
- `JWT_SECRET` harus sama dengan yang lama agar token login pengguna tetap
  valid. Kalau diganti, semua pengguna harus login ulang.
- Jangan aktifkan public port untuk Postgres.

Deploy, lalu cek log: harus muncul `database connected` dan
`split-bill-go running on port 8080`.

---

## 4. Deploy / arahkan frontend

Di environment frontend:

```env
PUBLIC_API_BASE_URL=https://api.splitkita.id
PUBLIC_GOOGLE_CLIENT_ID=<client id Google>
```

Lalu di Google Cloud Console → OAuth Client (Web) → **Authorized JavaScript
origins**, tambahkan origin frontend:

```
https://splitkita.id
```

Kalau frontend dan backend berada di origin yang sama lewat reverse proxy,
`PUBLIC_API_BASE_URL` boleh kosong dan `CORS_ALLOWED_ORIGINS` boleh kosong.

Kalau beda origin:

- `CORS_ALLOWED_ORIGINS` harus berisi origin frontend **persis** (skema +
  domain + port, tanpa trailing slash).
- Backend hanya mengizinkan origin yang terdaftar; `*` sengaja tidak didukung
  karena frontend mengirim `X-API-Key`.

---

## 5. Smoke test setelah cutover

1. `GET https://api.splitkita.id/health` → 200.
2. Buka frontend → dashboard menampilkan bill lama.
3. Buka satu bill lewat public link.
4. Login Google.
5. Tandai pembayaran (settlement) dan lihat riwayat.
6. Export CSV/PDF.
7. Reminder WhatsApp + QR invite.
8. Cek log backend tidak ada error koneksi DB.

Kalau langkah 1 gagal, cek `DB_HOST`, `DB_PASSWORD`, dan apakah backend
berada di network yang sama dengan Postgres.

---

## 6. Rollback

- **Jangan hapus** database lokal dan file di `backups/` sampai produksi
  terverifikasi.
- Kalau ada masalah, kembalikan env backend ke DB lokal dan jalankan backend
  lokal seperti sebelumnya.
- Simpan password DB lama & baru di password manager.

---

## 7. Backup

Aktifkan backup terjadwal di Coolify (menu Backups pada resource Postgres),
atau cron di host:

```bash
#!/usr/bin/env bash
set -euo pipefail
STAMP="$(date +%Y%m%d-%H%M%S)"
docker exec splitkita-db pg_dump -U splitkita -d splitkita -Fc \
  | gzip > "/var/backups/splitkita-$STAMP.dump.gz"
find /var/backups -name 'splitkita-*.dump.gz' -mtime +14 -delete
```

Simpan minimal 7–14 hari dan sesekali tes restore ke database percobaan.

---

## 8. Menambah database untuk web lain

Satu container PostgreSQL bisa menampung banyak database. Jalankan sebagai
superuser (sesuaikan nama container; user superuser Coolify biasanya user yang
dibuat saat resource dibuat, mis. `splitkita`):

```bash
docker exec -it <postgres-container> psql -U splitkita -d postgres \
  -c "CREATE ROLE web2_user LOGIN PASSWORD 'password-kuat';"

docker exec -it <postgres-container> psql -U splitkita -d postgres \
  -c "CREATE DATABASE web2 OWNER web2_user;"
```

`CREATE DATABASE` harus di perintah terpisah karena tidak boleh di dalam
transaksi. Dengan `OWNER`, `web2_user` otomatis punya akses ke schema `public`
(PostgreSQL 15+).

Jangan pakai initialization script untuk database yang sudah pernah start —
script itu hanya dieksekusi saat volume pertama kali dibuat.

---

## 9. Local development

Tetap pakai Postgres lokal (Docker Desktop) dengan database terpisah:

```powershell
docker start postgres
```

Jangan arahkan backend lokal ke DB production. Kalau butuh data mirip
production untuk debugging, restore dump ke database lokal baru:

```powershell
docker exec postgres psql -U postgres -d postgres -c "CREATE DATABASE splitkita_dev;"
docker cp backups\splitkita-<ts>.dump postgres:/tmp/restore.dump
docker exec postgres pg_restore -U postgres -d splitkita_dev --no-owner --no-privileges /tmp/restore.dump
```

---

## Checklist keamanan

- [ ] Public port Postgres mati.
- [ ] `.env` dan `backups/` tidak masuk git.
- [ ] Password DB hanya di Coolify secrets / password manager.
- [ ] Dashboard Coolify pakai password kuat + 2FA.
- [ ] `CORS_ALLOWED_ORIGINS` hanya origin frontend yang benar.
- [ ] Backup terjadwal aktif dan pernah dites restore.
