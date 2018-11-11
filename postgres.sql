CREATE TABLE users (
  name text PRIMARY KEY,
  email text NOT NULL,
  pk text
);

CREATE TABLE head (
  owner text NOT NULL references users (name),
  name text NOT NULL,
  cid text NOT NULL,
  updated_at timestamp NOT NULL DEFAULT now(),
  note text NOT NULL DEFAULT '',

  UNIQUE (owner, name),
  CHECK (owner ~ '[\w\d-]+'),
  CHECK (name ~ '[\w\d-]+')
);

CREATE INDEX ON head (owner);
CREATE INDEX ON head (name);
CREATE INDEX ON head (cid);

table users;
table head;
