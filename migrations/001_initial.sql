PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS operators (
 id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL, role TEXT NOT NULL,
 password_hash TEXT NOT NULL, disabled INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
 id TEXT PRIMARY KEY, operator_id TEXT NOT NULL REFERENCES operators(id), tenant_id TEXT NOT NULL,
 expires_at TEXT NOT NULL, revoked_at TEXT, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lots (
 id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, code TEXT NOT NULL, chemistry TEXT NOT NULL,
 state TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1, received_at TEXT NOT NULL,
 expires_at TEXT, hazard_score INTEGER NOT NULL DEFAULT 0, created_by TEXT NOT NULL,
 UNIQUE(tenant_id, code)
);
CREATE TABLE IF NOT EXISTS inspections (
 id TEXT PRIMARY KEY, lot_id TEXT NOT NULL REFERENCES lots(id), tenant_id TEXT NOT NULL,
 result TEXT NOT NULL, notes TEXT NOT NULL, inspector_id TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reservations (
 id TEXT PRIMARY KEY, lot_id TEXT NOT NULL REFERENCES lots(id), tenant_id TEXT NOT NULL,
 station TEXT NOT NULL, operator_id TEXT NOT NULL, state TEXT NOT NULL,
 created_at TEXT NOT NULL, released_at TEXT
);
CREATE TABLE IF NOT EXISTS recoveries (
 id TEXT PRIMARY KEY, lot_id TEXT NOT NULL REFERENCES lots(id), tenant_id TEXT NOT NULL,
 lithium_grams INTEGER NOT NULL, nickel_grams INTEGER NOT NULL, cobalt_grams INTEGER NOT NULL,
 state TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1, recovered_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS certificates (
 id TEXT PRIMARY KEY, lot_id TEXT NOT NULL REFERENCES lots(id), tenant_id TEXT NOT NULL,
 serial TEXT NOT NULL UNIQUE, state TEXT NOT NULL, published_at TEXT, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_events (
 id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, actor_id TEXT NOT NULL, object_type TEXT NOT NULL,
 object_id TEXT NOT NULL, action TEXT NOT NULL, result TEXT NOT NULL, request_id TEXT NOT NULL,
 payload TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS idempotency_keys (
 tenant_id TEXT NOT NULL, key TEXT NOT NULL, operation TEXT NOT NULL, response TEXT NOT NULL,
 created_at TEXT NOT NULL, PRIMARY KEY(tenant_id, key, operation)
);
CREATE TABLE IF NOT EXISTS outbox (
 id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, aggregate_id TEXT NOT NULL, event_type TEXT NOT NULL,
 payload TEXT NOT NULL, state TEXT NOT NULL, attempts INTEGER NOT NULL DEFAULT 0,
 next_attempt_at TEXT NOT NULL, last_error TEXT, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lots_tenant_state ON lots(tenant_id, state, received_at);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_events(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_state_time ON outbox(state, next_attempt_at);
