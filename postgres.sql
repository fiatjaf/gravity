CREATE TABLE users (
  name text PRIMARY KEY,
  email text NOT NULL,
  pk text
);

CREATE TABLE head (
  id serial PRIMARY KEY,
  owner text NOT NULL REFERENCES users (name),
  name text NOT NULL,
  cid text NOT NULL,
  updated_at timestamp NOT NULL DEFAULT now(),
  note text NOT NULL DEFAULT '',
  body text NOT NULL DEFAULT '',

  UNIQUE (owner, name),
  CONSTRAINT check_owner CHECK (owner ~ '[\w\d.-]+'),
  CONSTRAINT check_name CHECK (name ~ '[\w\d.-]+'),
  CONSTRAINT check_owner_size CHECK (character_length(owner) <= 35),
  CONSTRAINT check_name_size CHECK (character_length(name) <= 50),
  CONSTRAINT check_note_size CHECK (character_length(note) <= 280)
);

CREATE INDEX ON head (owner);
CREATE INDEX ON head (name);
CREATE INDEX ON head (cid);

CREATE TABLE history (
  id serial PRIMARY KEY,
  record_id int NOT NULL REFERENCES head(id) ON DELETE CASCADE,
  set_at timestamp NOT NULL DEFAULT now(),
  cid text NOT NULL,
  prev int,

  FOREIGN KEY (owner, name) REFERENCES head (owner, name) ON DELETE CASCADE
);

CREATE INDEX ON history (owner, name);

CREATE OR REPLACE FUNCTION update_history() RETURNS trigger AS $$
  DECLARE
    previous int;
  BEGIN
    IF TG_OP = 'UPDATE' THEN
      SELECT id INTO previous FROM history
        WHERE record_id = NEW.id
        ORDER BY id DESC LIMIT 1;
    END IF;

    INSERT INTO history (record_id, cid, prev)
      VALUES (NEW.id, NEW.cid, previous);

    RETURN NULL;
  END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_history_ins AFTER INSERT ON head
  FOR EACH ROW EXECUTE PROCEDURE update_history();
CREATE TRIGGER update_history_upd AFTER UPDATE OF cid ON head
  FOR EACH ROW WHEN (NEW.cid != OLD.cid) EXECUTE PROCEDURE update_history();

CREATE TABLE stars (
  source text NOT NULL REFERENCES users(name),
  target_owner text NOT NULL,
  target_name text NOT NULL,
  starred_at timestamp NOT NULL DEFAULT now(),

  FOREIGN KEY (target_owner, target_name) REFERENCES head (owner, name),
  UNIQUE (source, target_owner, target_name)
);

CREATE TABLE pub_user_followers (
  follower text NOT NULL,
  target text NOT NULL REFERENCES users (name),

  UNIQUE(follower, target)
);

CREATE VIEW pub_outbox AS
  SELECT
    owner,
    history.id::text AS raw_id,

    'Note' AS "type",
    set_at AS "published",
    owner || '/' || name || ': https://ipfs.io/ipfs/' || history.cid AS "content",
    'https://www.w3.org/ns/activitystreams#Public' AS "to"
  FROM history
  INNER JOIN head ON history.record_id = head.id
  ORDER BY history.set_at DESC;

table users;
table history;
table stars;
table pub_outbox;
table pub_user_followers;
