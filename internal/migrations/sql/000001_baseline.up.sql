-- Introduces the versioned migration ledger without changing the legacy schema.
-- Existing tables are still managed by AutoMigrate until their schema has been
-- captured in reviewed forward migrations.
SELECT 1;
