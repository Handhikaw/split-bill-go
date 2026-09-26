# App Split Bill – Product & Feature Concept (Revisi)

Dokumen ini adalah **konsep acuan resmi** untuk pengembangan App Split Bill versi 2. Dokumen ini digunakan sebagai **single source of truth** untuk diskusi UI, UX, backend (Golang), API design, dan monetisasi.

---

## 1. Tujuan Produk
Aplikasi untuk membantu pengguna membagi pengeluaran bersama secara adil dan transparan, baik untuk:
- Sekali transaksi (makan di restoran)
- Banyak transaksi dalam grup (liburan, nongkrong rutin)
- Patungan berbasis tujuan (beli kado, donasi kecil)

Fokus utama:
- Cepat
- Kolaboratif
- Tidak ribet login di awal

---

## 2. Fitur Utama (User Facing)

### 2.1 Split Bill (One-Time Bill)
Digunakan untuk satu tagihan satu pembayar

**Fitur:**
- OCR
- Input transaksi (item, harga, untuk siapa)
- Share link (kolaboratif, bisa edit)
- OIR / custom fee
- PPN
- Service Charge
- Diskon
- Multi currency
- Share via WhatsApp / link

**Batasan Free:**
- Tidak login
- Tidak bisa save database
- Tidak ada dashboard
- Ada iklan

---

### 2.2 Group Split Bill (Multi-Expense)
Digunakan untuk banyak transaksi dalam satu grup.

**Fitur:**
- Create group
- Share link (kolaboratif, bisa edit)
- Input transaksi:
  - siapa yang membayar
  - untuk siapa
  - nominal
- Hitung otomatis:
  - siapa utang ke siapa
  - ringkasan utang/piutang per user
- Save ke database

**Batasan Free:**
- Grup menjadi **read-only setelah 2 hari**
- Maksimal jumlah transaksi (mis. 15)
- Tidak bisa download PDF
- Ada iklan

---

### 2.3 Patungan (Goal-Based)
Digunakan untuk pengumpulan dana bersama.

**Fitur:**
- Buat grup patungan
- Share link penagihan
- Tandai pembayaran sudah dilakukan

**Batasan Free:**
- Maksimal peserta (mis. 5 orang)
- Data bertahan 1 bulan
- Ada iklan

---

## 3. Premium User (One-Time Purchase)
Premium berlaku untuk semua fitur.

### 3.1 Benefit Umum
- Tidak ada iklan
- Login (Google)
- Data bertahan selamanya
- Dark mode
- Export PDF & CSV
- Bisa edit ulang data lama

---

### 3.2 Split Bill (Premium)
- Semua fitur free
- Save ke database
- Download PDF
- Dashboard histori split bill

---

### 3.3 Group Split Bill (Premium)
- Semua fitur free
- Tidak ada batas waktu
- Tandai utang sudah lunas
- Dashboard:
  - daftar grup yang diikuti
  - ringkasan utang/piutang
  - grafik pengeluaran

---

### 3.4 Patungan (Premium)
- Semua fitur free
- Tidak ada batas waktu
- Unlimited peserta
- Dashboard & histori

---

## 4. Prinsip UX Penting
- Tidak wajib login di awal
- User harus **melihat hasil dulu**, baru ditawarkan premium
- Tidak ada data yang "hilang" secara paksa
- Free user selalu bisa melihat summary

---

## 5. Bahasa & Regional
- Bahasa: Indonesia & English
- Auto-detect dari browser
- Bahasa **bukan** fitur premium

---

## 6. Prinsip Teknis (Acuan Backend)
- Satu core logic untuk semua fitur (Expense Engine)
- Entitas utama:
  - User (optional / anonymous)
  - Group
  - Expense
  - Participant
  - Settlement
  - Payment Status
  - Subscription

Backend akan dikembangkan menggunakan **Golang**, tanpa auto-migration, dengan database schema eksplisit.

---

## 7. Scope MVP

### MVP v1
- Split Bill (anonymous)
- Group Split Bill (basic)
- Share link
- Hitung utang otomatis
- Save DB
- PDF export (simple)

### MVP v1.1
- Login Google
- Dashboard
- Premium toggle

### v2
- Payment gateway
- Grafik
- Patungan lanjutan

---

Dokumen ini menjadi dasar semua diskusi lanjutan terkait UI, API, database, dan flow aplikasi.

