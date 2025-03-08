CREATE TABLE IF NOT EXISTS alerts (
  id SERIAL PRIMARY KEY,
  node_id INT NOT NULL,
  metric TEXT NOT NULL,
  duration INT NOT NULL,
  threshold float DEFAULT 0,
  net_rece_threshold float DEFAULT 0,
  net_send_threshold float DEFAULT 0,
  email text,
  discord_webhook text,
  slack_webhook text,
  is_active boolean DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);