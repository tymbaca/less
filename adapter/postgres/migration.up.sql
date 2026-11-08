CREATE SCHEMA IF NOT EXISTS less;

CREATE TABLE IF NOT EXISTS less.keys (
    "key"      TEXT PRIMARY KEY,
    "node"     TEXT NOT NULL,
    "deadline" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS less.nodes (
    "node"      TEXT PRIMARY KEY,
    "deadline"  TIMESTAMPTZ
);

-- TODO: comments
