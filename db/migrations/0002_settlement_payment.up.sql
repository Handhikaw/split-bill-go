-- Ledger pelunasan (mark-as-paid) untuk group & split bill.
-- Jalankan manual sebelum deploy backend yang memakai fitur ini.

CREATE TABLE IF NOT EXISTS settlement_payment (
  id                  bigserial PRIMARY KEY,
  bill_id             bigint NOT NULL REFERENCES bill(id) ON DELETE CASCADE,
  from_participant_id bigint NOT NULL REFERENCES participant(id) ON DELETE RESTRICT,
  to_participant_id   bigint NOT NULL REFERENCES participant(id) ON DELETE RESTRICT,
  amount              bigint NOT NULL CHECK (amount > 0),
  note                varchar(255),
  paid_at             timestamptz NOT NULL DEFAULT now(),
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CHECK (from_participant_id <> to_participant_id)
);

CREATE INDEX IF NOT EXISTS idx_settlement_payment_bill ON settlement_payment(bill_id);
CREATE INDEX IF NOT EXISTS idx_settlement_payment_from ON settlement_payment(from_participant_id);
CREATE INDEX IF NOT EXISTS idx_settlement_payment_to ON settlement_payment(to_participant_id);
