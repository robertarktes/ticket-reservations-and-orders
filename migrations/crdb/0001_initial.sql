CREATE DATABASE IF NOT EXISTS tro;

SET database = tro;

CREATE TABLE events (
  id UUID PRIMARY KEY,
  title TEXT,
  starts_at TIMESTAMPTZ
);

CREATE TABLE seats (
  event_id UUID,
  seat_no TEXT,
  PRIMARY KEY (event_id, seat_no)
);

CREATE TABLE holds (
  id UUID PRIMARY KEY,
  event_id UUID,
  seat_no TEXT,
  user_id UUID,
  expires_at TIMESTAMPTZ,
  status TEXT CHECK (status IN ('ACTIVE', 'EXPIRED', 'RELEASED')),
  UNIQUE (event_id, seat_no) WHERE status = 'ACTIVE'
);

CREATE TABLE orders (
  id UUID PRIMARY KEY,
  user_id UUID,
  status TEXT CHECK (status IN ('PENDING', 'CONFIRMED', 'FAILED')),
  total_amount NUMERIC,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE order_items (
  order_id UUID,
  event_id UUID,
  seat_no TEXT,
  price NUMERIC,
  PRIMARY KEY (order_id, event_id, seat_no)
);

CREATE TABLE outbox (
  id UUID PRIMARY KEY,
  aggregate_type TEXT,
  aggregate_id UUID,
  event_type TEXT,
  payload_json JSONB,
  created_at TIMESTAMPTZ DEFAULT now(),
  published_at TIMESTAMPTZ,
  status TEXT CHECK (status IN ('NEW', 'PUBLISHED', 'FAILED')),
  dedupe_key TEXT UNIQUE
);

CREATE TABLE inbox (
  message_id TEXT PRIMARY KEY,
  received_at TIMESTAMPTZ
);