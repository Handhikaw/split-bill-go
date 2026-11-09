-- Backfill identitas bill/owner setelah fix middleware auth (X-API-Key).
-- Sebelumnya bill.created_by NULL karena route create memakai OptionalAuth
-- (Authorization Bearer), sedangkan frontend mengirim X-API-Key.

UPDATE bill b
SET created_by = p.user_id
FROM participant p
WHERE p.bill_id = b.id
  AND p.is_owner = true
  AND p.user_id IS NOT NULL
  AND b.created_by IS NULL;

UPDATE participant p
SET user_id = b.created_by
FROM bill b
WHERE p.bill_id = b.id
  AND p.is_owner = true
  AND p.user_id IS NULL
  AND b.created_by IS NOT NULL;

-- Satu identitas Google hanya boleh dipakai satu user.
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_oauth
  ON users (oauth_provider, oauth_id)
  WHERE oauth_id <> '';
