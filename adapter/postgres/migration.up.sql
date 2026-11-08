CREATE SCHEMA IF NOT EXISTS less;

CREATE TABLE IF NOT EXISTS less.keys (
    "key"      TEXT PRIMARY KEY,
    "node"     TEXT NOT NULL,
    "deadline" TIMESTAMPTZ
);

COMMENT ON TABLE less.keys IS 'Leader election records: one row per job key';
COMMENT ON COLUMN less.keys.key IS 'Job key the candidates compete for';
COMMENT ON COLUMN less.keys.node IS 'ID of the node that currently holds the leadership';
COMMENT ON COLUMN less.keys.deadline IS 'Moment when the leadership expires unless renewed';

CREATE TABLE IF NOT EXISTS less.nodes (
    "node"      TEXT PRIMARY KEY,
    "deadline"  TIMESTAMPTZ
);

COMMENT ON TABLE less.nodes IS 'Liveness records used by the balancer: one row per node';
COMMENT ON COLUMN less.nodes.node IS 'ID of the node';
COMMENT ON COLUMN less.nodes.deadline IS 'Moment after which the node is considered dead';
