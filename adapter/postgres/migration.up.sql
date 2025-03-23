CREATE SCHEMA IF NOT EXISTS less;
CREATE TABLE IF NOT EXISTS less.record (
    "key"      TEXT PRIMARY KEY,
    "value"    TEXT NOT NULL,
    "deadline" TIMESTAMPTZ
);
