-- Patungan (goal bill) schema.
-- Jalankan manual sebelum deploy backend yang memakai fitur ini.
-- Tidak ada auto-migrate di aplikasi.

-- 1) Jenis bill: split | group | goal.
--    Sebelumnya jenis ditebak dari expense.type; kolom ini membuatnya eksplisit.
ALTER TABLE bill ADD COLUMN IF NOT EXISTS kind varchar(10) NOT NULL DEFAULT 'group';

UPDATE bill b
SET kind = 'split'
WHERE b.kind = 'group'
  AND EXISTS (
    SELECT 1 FROM expense e
    WHERE e.bill_id = b.id AND e.type = 'SPLIT_BILL'
  );

-- 2) Metadata patungan (1:1 dengan bill).
CREATE TABLE IF NOT EXISTS goal_bill (
  id            bigserial PRIMARY KEY,
  bill_id       bigint NOT NULL UNIQUE REFERENCES bill(id) ON DELETE CASCADE,
  target_amount bigint NOT NULL CHECK (target_amount > 0),
  description   varchar(255),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

-- 3) Ledger kontribusi (uang yang sudah diterima).
--    RESTRICT: peserta yang sudah kontribusi tidak boleh dihapus sebelum
--    kontribusinya dihapus, supaya total tidak berubah diam-diam.
CREATE TABLE IF NOT EXISTS contribution (
  id             bigserial PRIMARY KEY,
  bill_id        bigint NOT NULL REFERENCES bill(id) ON DELETE CASCADE,
  participant_id bigint NOT NULL REFERENCES participant(id) ON DELETE RESTRICT,
  amount         bigint NOT NULL CHECK (amount > 0),
  note           varchar(255),
  paid_at        timestamptz NOT NULL DEFAULT now(),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contribution_bill ON contribution(bill_id);
CREATE INDEX IF NOT EXISTS idx_contribution_participant ON contribution(participant_id);
