CREATE UNIQUE INDEX idx_addresses_one_default_per_user ON addresses (user_id) WHERE is_default;
