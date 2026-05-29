-- Reversal of migration 000001.
-- Removes the pgcrypto extension.
-- Note: only safe to run if no tables using gen_random_uuid() exist.
DROP EXTENSION IF EXISTS "pgcrypto";