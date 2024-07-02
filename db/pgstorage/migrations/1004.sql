-- +migrate Down

ALTER TABLE sync.deposit DROP COLUMN IF EXISTS dest_contract_addr;

-- +migrate Up

CREATE TABLE IF NOT EXISTS sync.bridge_balance
(
    id SERIAL PRIMARY KEY,
    original_token_addr VARCHAR,
    network_id INTEGER,
    balance VARCHAR,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    modify_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT bridge_balance_uidx UNIQUE (original_token_addr, network_id)
);
