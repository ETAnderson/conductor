CREATE TABLE IF NOT EXISTS run_channel_results (
  run_id VARCHAR(64) NOT NULL,
  tenant_id BIGINT UNSIGNED NOT NULL,
  channel VARCHAR(64) NOT NULL,
  attempt INT NOT NULL,
  ok_count INT NOT NULL,
  err_count INT NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (run_id, channel),
  KEY idx_tenant_run (tenant_id, run_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS run_channel_items (
  run_id VARCHAR(64) NOT NULL,
  channel VARCHAR(64) NOT NULL,
  product_key VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL,
  message VARCHAR(255) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (run_id, channel, product_key),
  KEY idx_run_channel (run_id, channel)
) ENGINE=InnoDB;
