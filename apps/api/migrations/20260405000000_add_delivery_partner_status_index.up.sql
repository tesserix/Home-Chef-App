-- Index supporting: SELECT id FROM deliveries WHERE delivery_partner_id = ? AND status IN (...)
-- Used by UpdateLocation handler on every GPS update to look up the active delivery.
CREATE INDEX IF NOT EXISTS idx_deliveries_partner_status
    ON deliveries (delivery_partner_id, status);
