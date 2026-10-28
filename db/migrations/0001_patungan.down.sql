DROP INDEX IF EXISTS idx_contribution_participant;
DROP INDEX IF EXISTS idx_contribution_bill;
DROP TABLE IF EXISTS contribution;
DROP TABLE IF EXISTS goal_bill;
ALTER TABLE bill DROP COLUMN IF EXISTS kind;
