-- awards
CREATE TABLE IF NOT EXISTS awards (
  id UUID PRIMARY KEY,
  event_id UUID NOT NULL UNIQUE,
  user_id TEXT NOT NULL,
  stake NUMERIC NOT NULL,
  bonus_amount NUMERIC NOT NULL,
  offer_id INT NOT NULL REFERENCES offers(id),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

