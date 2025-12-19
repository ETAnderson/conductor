-- Tenants (multi-tenant boundary)
CREATE TABLE IF NOT EXISTS tenants (
  tenant_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (tenant_id),
  UNIQUE KEY uq_tenants_name (name)
) ENGINE=InnoDB;

-- Feeds belong to a tenant
CREATE TABLE IF NOT EXISTS feeds (
  feed_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (feed_id),
  KEY idx_feeds_tenant (tenant_id),
  CONSTRAINT fk_feeds_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
) ENGINE=InnoDB;

-- Canonical product state (what we compare against for delta detection)
CREATE TABLE IF NOT EXISTS product_state (
  tenant_id BIGINT UNSIGNED NOT NULL,
  product_key VARCHAR(255) NOT NULL,
  normalized_hash CHAR(64) NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (tenant_id, product_key),
  KEY idx_product_state_tenant_updated (tenant_id, updated_at)
) ENGINE=InnoDB;

-- Runs (one ingestion operation)
CREATE TABLE IF NOT EXISTS runs (
  run_id VARCHAR(64) NOT NULL,
  tenant_id BIGINT UNSIGNED NOT NULL,
  feed_id BIGINT UNSIGNED NULL,
  status VARCHAR(64) NOT NULL,
  push_triggered TINYINT(1) NOT NULL,
  received INT NOT NULL,
  valid INT NOT NULL,
  rejected INT NOT NULL,
  unchanged INT NOT NULL,
  enqueued INT NOT NULL,
  warnings_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (run_id),
  KEY idx_runs_tenant_created (tenant_id, created_at),
  KEY idx_runs_feed_created (feed_id, created_at),
  CONSTRAINT fk_runs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id),
  CONSTRAINT fk_runs_feed FOREIGN KEY (feed_id) REFERENCES feeds(feed_id)
) ENGINE=InnoDB;

-- Per-product results for a run (cheap reporting + debugging)
CREATE TABLE IF NOT EXISTS run_products (
  run_id VARCHAR(64) NOT NULL,
  product_key VARCHAR(255) NOT NULL,
  disposition VARCHAR(64) NOT NULL,
  reason VARCHAR(255) NULL,
  normalized_hash CHAR(64) NULL,
  issues_json JSON NULL,
  PRIMARY KEY (run_id, product_key),
  KEY idx_run_products_run (run_id),
  CONSTRAINT fk_run_products_run FOREIGN KEY (run_id) REFERENCES runs(run_id)
) ENGINE=InnoDB;

-- Idempotency cache (per tenant + endpoint)
CREATE TABLE IF NOT EXISTS idempotency (
  tenant_id BIGINT UNSIGNED NOT NULL,
  endpoint VARCHAR(255) NOT NULL,
  idem_key_hash CHAR(64) NOT NULL,
  status_code INT NOT NULL,
  response_body_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  PRIMARY KEY (tenant_id, endpoint, idem_key_hash),
  KEY idx_idem_expires (expires_at),
  CONSTRAINT fk_idem_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
) ENGINE=InnoDB;
