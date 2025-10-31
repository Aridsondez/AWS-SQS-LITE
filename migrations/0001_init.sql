-- 0001_init.sql
-- Initial schema for SQS-Lite

-- Messages live in a single table, partitioned logically by `queue`.
-- A message is available when:
--   lease_until IS NULL  AND  not_before <= now()
-- When claimed, we set:
--   lease_until = now() + <visibility>
-- and increment delivery_count. On ack, we DELETE the row.

CREATE TABLE IF NOT EXISTS messages (
  id               BIGSERIAL PRIMARY KEY,
  queue            TEXT        NOT NULL,                  -- logical queue name
  body             JSONB       NOT NULL,                  -- payload
  enqueued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),    -- enqueue time
  not_before       TIMESTAMPTZ NOT NULL DEFAULT now(),    -- delay/schedule
  lease_until      TIMESTAMPTZ,                           -- NULL => available; set when claimed
  delivery_count   INT         NOT NULL DEFAULT 0,        -- attempts so far
  max_retries      INT         NOT NULL DEFAULT 5,        -- DLQ threshold
  dlq              TEXT,                                  -- optional DLQ queue name
  trace_id         TEXT,                                  -- optional tracing id

  -- Basic sanity checks
  CONSTRAINT chk_delivery_count_nonneg CHECK (delivery_count >= 0),
  CONSTRAINT chk_max_retries_nonneg    CHECK (max_retries   >= 0)
);

-- Fast path for receivers:
-- Partial index only on "not leased"; planner will still use not_before for filtering <= now().
CREATE INDEX IF NOT EXISTS idx_messages_available
  ON messages (queue, not_before, id)
  WHERE lease_until IS NULL;

-- For the sweeper to find in-flight messages efficiently (expired check is done at query time):
CREATE INDEX IF NOT EXISTS idx_messages_inflight
  ON messages (queue, lease_until)
  WHERE lease_until IS NOT NULL;

-- General stats/ops by queue:
CREATE INDEX IF NOT EXISTS idx_messages_queue ON messages (queue);
