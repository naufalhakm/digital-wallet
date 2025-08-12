DROP TRIGGER IF EXISTS update_wallets_updated_at ON wallets;
DROP INDEX IF EXISTS idx_wallets_user_id;
DROP TABLE IF EXISTS wallets CASCADE;
