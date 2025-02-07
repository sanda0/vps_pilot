CREATE TABLE IF NOT EXISTS node_sys_info (
  id SERIAL PRIMARY KEY,
  node_id INTEGER NOT NULL,
  os TEXT,
  platform TEXT,
  platform_version TEXT,
  kernel_version TEXT,
  cpus int,
  total_memory float,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE
);