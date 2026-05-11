DROP POLICY IF EXISTS tenant_isolation_rate_cards ON messaging.rate_cards;
DROP POLICY IF EXISTS tenant_isolation_wallet_ledger ON messaging.wallet_ledger;

DROP INDEX IF EXISTS messaging.idx_rate_cards_lookup;
DROP INDEX IF EXISTS messaging.idx_wallet_ledger_reference;
DROP INDEX IF EXISTS messaging.idx_wallet_ledger_tenant_currency;

DROP TABLE IF EXISTS messaging.rate_cards;
DROP TABLE IF EXISTS messaging.wallet_ledger;
