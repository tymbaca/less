-- less_keys holds leader election records: one row per job key.
--   key      - job key the candidates compete for
--   node     - id of the node that currently holds the leadership
--   deadline - moment when the leadership expires unless renewed
CREATE TABLE IF NOT EXISTS less_keys (
    "key"      text NOT NULL,
    "node"     text NOT NULL,
    "deadline" datetime, 
    PRIMARY KEY (key)
);

-- less_nodes holds liveness records used by the balancer: one row per node.
--   node     - id of the node
--   deadline - moment after which the node is considered dead
CREATE TABLE IF NOT EXISTS less_nodes (
    "node"     text NOT NULL,
    "deadline" datetime, 
    PRIMARY KEY (node)
);
