# AGENT.md — split-bill-go Backend

> Baca file ini dulu sebelum kerja. Jangan baca ulang semua folder kalau sudah ada di sini.

## 1. Gambaran Umum
Backend Split Bill (Go) — patungan sekali bayar (Split Bill) + multi-expense grup (Group Bill).
- Konsep produk lengkap: `CONCEPT.md` (single source of truth fitur Free vs Premium).
- Entry point: `cmd/api/main.go`
- Module: `github.com/harkandavez/split-bill-go`, Go `1.25.5`

## 2. Tech Stack
- HTTP: `gin-gonic/gin v1.11.0` (`gin.Default()` + `ErrorMiddleware`)
- DB: Postgres + `gorm.io/gorm` + `gorm.io/driver/postgres` — **tanpa auto-migrate**, schema eksplisit di DB
- Decimal/uamg: `shopspring/decimal` (hanya di `settled_debt`), mayoritas uang = `int64` smallest unit
- ID publik share-link: `go-nanoid/v2` (20 char) -> kolom `bill.public_id`
- Auth: `golang-jwt/jwt/v5`, anon via `google/uuid`, config via `joho/godotenv`

## 3. Cara Jalan & Config
```bash
go run ./cmd/api/main.go
# atau
go build ./... && ./split-bill-go
```
- Config: `internal/config/config.go` -> `Load()` baca `.env` (fallback ke env sistem).
- Env yang dipakai: `APP_NAME, APP_PORT, APP_ENV, JWT_SECRET, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME`
- DSN: `internal/config/database.go` (`InitDB`, Postgres, tanpa `sslmode`).
- **Gotcha:** `main.go:27` hardcode `r.Run(":8080")`, `cfg.AppPort` hanya untuk log. `gin.ReleaseMode` hanya jika `APP_ENV=production`.

## 4. Struktur Folder
```
cmd/api/main.go                  # wiring gin + DB + router
internal/router/router.go        # DI manual + semua route (baca ini dulu untuk API)
internal/config/                 # config.go, database.go
internal/middleware/             # auth_middleware.go (ParticipantMiddleware)
internal/handler/                # gin handler: bll, expense, participant, payment, user, health, error
internal/service/                # interface *_service.go + impl *_service_impl.go
internal/repository/             # interface + impl per agregat + group_bill_repository.go
internal/model/entity/           # struct GORM (nama tabel eksplisit)
internal/model/dto/              # request/response JSON
internal/model/api_response.go   # Success() / Created()
internal/model/api_error.go      # ApiError + NewBadRequest/NotFound/InternalError
internal/helper/money/           # money.go, money_test.go
internal/helper/money/split/     # split.go (int64), split_test.go
internal/helper/engine/          # engine.go (core split logic), engine_test.go
```

## 5. Layer & Pola Wajib
1. **Handler** (`internal/handler/*`): `ShouldBindJSON` -> panggil service -> `c.Error(err)` kalau gagal -> `c.JSON(..., model.Success/Created)`. Jangan `c.JSON` error langsung.
2. **ErrorMiddleware** (`handler/error_handler.go`): `errors.As` ke `*model.ApiError` -> render `{code, message}` dengan `HttpStatus`. Error lain -> 500.
3. **Service**: validasi bisnis + transaksi GORM + panggil `engine` + mapping `Money.String()`. Interface di `*_service.go`, impl di `*_service_impl.go`.
4. **Repository**: query tipis GORM per agregat. Cek `gorm.ErrRecordNotFound` -> ubah ke `NewNotFound` di service.
5. **DI manual** di `router.go:12-34`: repo -> service -> handler. Kalau tambah dependency baru, daftarkan di sini.
6. **Response standar**: sukses `{code:"OK"/"Created", message:"success", data:...}`, gagal `{code:"BAD_REQUEST"|"NOT_FOUND"|"INTERNAL_SERVER_ERROR", message:...}`.

## 6. API Routes (`internal/router/router.go`)
```
GET  /health
POST /api/split-bills                  # OptionalAuth + auto-create owner participant {title, currency, creator_name?} -> {public_id, owner_participant_id}
GET  /api/split-bills/:public_id       # GetSplitBill (asumsi 1 expense)
POST /api/group-bills                  # OptionalAuth + auto-create owner participant -> {public_id,title,status,expired_at,owner_participant_id}
GET  /api/group-bill/:public_id        # GetGroupBill (note: singular "group-bill")
POST /api/split-bills/:public_id/join  # JoinSplitBill {name, user_id?}
PUT  /api/payment/:public_id           # ProcessPayment {expense_id, participant_id, amount string, old_participant_id?} -> upsert (expense_id,paid_by); bila old_participant_id beda = pindah payer atomik (hapus lama + upsert baru), 404 bila payment lama tidak ada
POST /api/auth/anonymous               # header X-Anonymous-Id: UUID -> {access_token, user_id, role}
POST /api/auth/google                  # stub 501 NOT_IMPLEMENTED (UserHandler.AuthGoogle)

# butuh header X-Participant-ID (optional, dimasukkan ke ctx "participant_id"):
PUT    /api/participant/:public_id
DELETE /api/participant/:public_id/:id
GET    /api/expense/category
PUT    /api/expense/category
POST   /api/expense
PATCH  /api/expense/:public_id/:expense_id
DELETE /api/expense/:public_id/:expense_id
POST   /api/expense/:public_id/:expense_id/item
PATCH  /api/expense/:public_id/:expense_id/item/:item_id
DELETE /api/expense/:public_id/:expense_id/item/:item_id
```
- `ParticipantMiddleware` hanya parse header, **tidak blokir** kalau kosong/invalid.

## 7. Entity / Tabel
| Entity | Tabel | Catatan |
|---|---|---|
| `Bill` | `bill` | `public_id unique`, `currency(3)`, `status`, `created_by(user)`, `expired_at` |
| `Expense` | `expense` | `bill_id`, `type` (`SPLIT_BILL`/grup), `amount/tax/discount/charge int64`, `category_id?`, `created_by(participant)` |
| `ExpenseItem` | `expense_item` | `expense_id`, `name`, `quantity`, `price int64` |
| `Participant` | `participant` | `bill_id`, `name`, `user_id?`, `is_owner` |
| `ExpenseParticipant` | `expense_participant` | `expense_id, participant_id, share_amount, share_type` |
| `ExpenseItemParticipant` | `expense_item_participant` | `item_id, participant_id, share_amount` (tanpa timestamps) |
| `Payment` | `payment` | `bill_id, expense_id, paid_by(participant), amount, paid_at` (nama file typo: `payement.go`) |
| `ActivityLog` | `activity_logs` | `bill_id, participant_id, action_type, target_name?, amount?, metadata jsonb` |
| `Category` | `category` | `name, icon, color_code, is_default, user_id?` |
| `SettledDebt` | `settled_debt` | `bill_id, expense_id?, from/to participant, amount numeric(12,2), status` — belum dipakai engine |
| `User` | `users` | `username, email unique, role (ANONYMOUS/PREMIUM...), anon_id, oauth_*` |

## 8. Core Logic (jangan duplikasi)
- **`helper/money/money.go`**: `Money{Amount int64, Currency string}` smallest unit. Scale: `IDR:0, JPY:0, USD:2, EUR:2, KWD:3`, currency lain **panic**. `FromString/FromFloat`, `Add/Sub/Multiply`, `Split(n)` (sisa ke index awal), `SplitRandom`, `ApplyTax/ServiceCharge/Discount*` (round Half-Up), `String()` format sesuai scale.
- **`helper/money/split/split.go`**: versi `int64` mentah dari `SplitRandom`.
- **`helper/engine/engine.go`**:
  - `CalculateBalances(participants, expenses, items, itemParticipants, expenseParticipants, payments, currency)`: kredit dari `payments` (`paid_by`, agregat level bill — `ExpenseID` payment diabaikan by design). Debit: expense ber-item → dari `itemParticipants.share_amount`; expense tanpa item (SIMPLE) → dari `expenseParticipants.share_amount` (keduanya tidak pernah digabung — summary table diabaikan bila ada item, anti double-count). Share milik participant yang sudah dihapus di-skip. `tax+charge-discount` dibagi proporsional (rounding `math.Round`, sisa ke partisipan terakhir by ID sort).
  - `GenerateSettlements(balances)`: greedy debtor->creditor, sort deterministik, return `dto.Settlement{from,to,amount string}`. Settlement parsial by design bila ada expense belum dibayar (total net ≠ 0) — validasi net-zero sengaja mati.
  - Validasi total-net-zero saat ini **dikomentari mati**.
- Test acuan: `money_test.go`, `split/split_test.go`, `engine/engine_test.go` — jalankan `go test ./...` sebelum ubah engine/money.

## 9. Aturan Bisnis Penting
- Split Bill = 1 `Bill` + 1 `Expense{type:SPLIT_BILL}` (dibuat di `bill_service_impl.go:38-62`).
- Group Bill free: `ExpiredAt = now+48h`, premium (`user.role==PREMIUM` case-insensitive): `nil` (kekal). Lihat `CreateGroupBill`.
- `AuthAnon`: wajib header `X-Anonymous-Id` UUID valid; cari/buat `User{role:ANONYMOUS}`; JWT HS256 `user_id, role`, exp 30 hari. `jwtSecret` di-inject dari `cfg.JWTSecret` via `router.RegisterRoutes(r, db, cfg)`.
- Identitas create bill: `OptionalAuthMiddleware(cfg.JWTSecret)` di `POST /split-bills` & `POST /group-bills` — ada Bearer valid → context `auth_user_id`/`auth_role`, tanpa token → guest. Handler baca via `middleware.AuthUserID(c)` → service pakai langsung sebagai `bill.created_by`. Body `created_by` sudah **dihapus** dari `CreateSpiltBillRq`/`CreateGroupBillRq` (tidak ada lagi jalur spoofing).
- Bedakan anon: lookup `users` by effective id → `role=="ANONYMOUS"` atau `anon_id!=""`. Anon/terdaftar/guest **semua** auto-create `participant{is_owner:true}` dalam 1 transaksi GORM dengan bill (+expense untuk split bill). Premium → `ExpiredAt=nil`, selain itu +48 jam.
- Nama owner: `req.creator_name` (trim, max 100) → `users.username` → `"Owner"` (`resolveOwnerName`). Anon bawaan `username` kosong jadi fallback `"Owner"`.
- Amount di DTO selalu **string** (`"25000"`, `"10.00"` sesuai currency) — parse via `money.FromString`, format via `Money.String()`.
- `SyncExpenseTotal` ada di `ExpenseService` — panggil setelah ubah item agar `expense.amount` konsisten.

## 10. Bug / Tech-Debt (sudah fixed 2026-09-23, tersisa catatan)
- [fixed] `GetSplitBill` panic `expenses[0]` -> sekarang cek `len(expenses)==0` + handle semua repo/engine error; `Tax/Charge/Discount` sudah baca field masing-masing; slice kosong dikembalikan sebagai `[]` bukan `null`.
- [fixed] `main.go` hardcode `:8080` -> sekarang pakai `cfg.AppPort` (fallback `8080`).
- [fixed] `NewUserService(userRepo, "")` -> sekarang inject `cfg.JWTSecret`; fallback `os.Getenv` dihapus, error jelas kalau kosong.
- [fixed] `POST /api/auth/google` tanpa handler (bikin gin panic saat start) -> sekarang ada stub `UserHandler.AuthGoogle` return `501 NOT_IMPLEMENTED` (`model.NewNotImplemented`).
- [fixed] `fmt.Println` debug di `engine.go` & `auth_middleware.go` -> dihapus.
- [fixed] `repository/group_bill_repository.go` dead code (interface tak terpakai) -> dihapus.
- [fixed] `entity/payement.go` typo -> rename ke `entity/payment.go`.
- [fixed] `Money.String()` negatif berkoma (`-10.05` jadi `"-10.-5"` di USD) -> format dari nilai absolut + prefix `-`. Test di `money_format_test.go`.
- [fixed] currency asing bikin `money.New` panic -> `money.IsSupportedCurrency` + 400 di create bill.
- [fixed] `CreateExpenseItem` tanpa peserta panic divide-by-zero -> tolak 400.
- [fixed] `ProcessPayment` terima amount nol/negatif + format salah jadi 500 -> tolak 400 bila tidak positif; format salah jadi 400.
- [fixed] `settlements: null` di GetGroupBill -> koersi ke `[]`.
- [fixed] activity log FK violation (`fk_participant`): `X-Participant-ID` ngarang/tidak se-bill lolos ke `activity_logs` -> 200 + error async. Sekarang actor divalidasi (`ExistsByIDAndBillID`, 0 = tanpa header -> skip log) di participant + semua mutasi expense sebelum transaksi; self-delete (keluar grup) log tanpa actor agar tidak langgar FK. `UpsertParticipant` tanpa `is_owner` tadinya panic nil-deref -> default false.
- [sisa] `engine_test.go` lama seluruhnya dikomentari mati -> digantikan `engine_balance_test.go` (SIMPLE, anti double-count, ghost share, diskon) yang aktif.

## 11. Cara Tambah Fitur (checklist)
1. Entity di `model/entity/` + DTO di `model/dto/`.
2. Repository interface + impl.
3. Service interface + impl (error pakai `model.New*`, uang pakai `money`).
4. Handler pakai pola `ShouldBindJSON` + `c.Error`.
5. Daftarkan DI + route di `router.go`.
6. `go vet ./... && go test ./...`, pastikan JSON response ikut pola `ApiResponse`.
