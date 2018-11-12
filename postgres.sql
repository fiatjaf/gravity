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
  note text,

  UNIQUE (owner, name),
  CONSTRAINT check_owner CHECK (owner ~ '[\w\d.-]+'),
  CONSTRAINT check_name CHECK (name ~ '[\w\d.-]+'),
  CONSTRAINT check_owner_size CHECK (character_length(owner) <= 35),
  CONSTRAINT check_name_size CHECK (character_length(name) <= 50),
  CONSTRAINT check_note_size CHECK (character_length(note) <= 280),
);

CREATE INDEX ON head (owner);
CREATE INDEX ON head (name);
CREATE INDEX ON head (cid);

table users;
table head;
