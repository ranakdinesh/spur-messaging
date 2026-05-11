-- sql/queries/billing.sql

-- name: CreateWalletLedgerEntry :one
INSERT INTO messaging.wallet_ledger (
    id, tenant_id, entry_type, amount, currency, channel, category,
    reference_type, reference_id, description, metadata, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: ListWalletLedgerEntries :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.wallet_ledger
WHERE tenant_id = $1
AND currency = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetWalletBalance :one
SELECT
    $1::uuid AS tenant_id,
    $2::text AS currency,
    COALESCE(SUM(CASE
        WHEN entry_type IN ('credit', 'refund', 'adjustment') THEN amount
        WHEN entry_type = 'debit' THEN -amount
        ELSE 0
    END), 0)::numeric AS current_balance,
    COALESCE(SUM(CASE
        WHEN entry_type = 'hold' THEN amount
        WHEN entry_type = 'release' THEN -amount
        ELSE 0
    END), 0)::numeric AS reserved_balance,
    (
        COALESCE(SUM(CASE
            WHEN entry_type IN ('credit', 'refund', 'adjustment') THEN amount
            WHEN entry_type = 'debit' THEN -amount
            ELSE 0
        END), 0)
        -
        COALESCE(SUM(CASE
            WHEN entry_type = 'hold' THEN amount
            WHEN entry_type = 'release' THEN -amount
            ELSE 0
        END), 0)
    )::numeric AS available_balance,
    COALESCE(MAX(created_at), now())::timestamptz AS updated_at
FROM messaging.wallet_ledger
WHERE tenant_id = $1
AND currency = $2;

-- name: WalletLedgerReferenceExists :one
SELECT EXISTS (
    SELECT 1 FROM messaging.wallet_ledger
    WHERE tenant_id = $1
      AND reference_type = $2
      AND reference_id = $3
      AND entry_type = $4
);

-- name: CreateRateCard :one
INSERT INTO messaging.rate_cards (
    id, tenant_id, channel, category, country, currency, unit_price, effective_from, effective_to
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetActiveRateCard :one
SELECT *
FROM messaging.rate_cards
WHERE channel = $2
  AND category = $3
  AND country = $4
  AND currency = $5
  AND effective_from <= $6
  AND (effective_to IS NULL OR effective_to > $6)
  AND (tenant_id = $1 OR tenant_id IS NULL)
ORDER BY tenant_id NULLS LAST, effective_from DESC
LIMIT 1;
