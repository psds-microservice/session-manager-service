CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS consultation_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL,
  pin VARCHAR(12) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'waiting',
  lead_operator_id UUID,
  stream_session_id UUID,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP WITH TIME ZONE,
  deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_consultation_sessions_pin ON consultation_sessions(pin) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_consultation_sessions_client_id ON consultation_sessions(client_id);
CREATE INDEX IF NOT EXISTS idx_consultation_sessions_status ON consultation_sessions(status);
