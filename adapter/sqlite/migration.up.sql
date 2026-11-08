CREATE TABLE IF NOT EXISTS less_keys (
    "key"      text NOT NULL,
    "node"     text NOT NULL,
    "deadline" datetime, 
    PRIMARY KEY (key)
);

CREATE TABLE IF NOT EXISTS less_nodes (
    "node"     text NOT NULL,
    "deadline" datetime, 
    PRIMARY KEY (node)
);

-- TODO: comments
