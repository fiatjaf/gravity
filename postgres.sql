CREATE TABLE head (
  owner text NOT NULL,
  name text NOT NULL,
  cid text NOT NULL,
  updated_at timestamp NOT NULL DEFAULT now(),

  UNIQUE (owner, name),
  CHECK (owner ~ '[\w\d-]+'),
  CHECK (name ~ '[\w\d-]+')
);

CREATE INDEX ON head (owner);
CREATE INDEX ON head (name);
CREATE INDEX ON head (cid);
