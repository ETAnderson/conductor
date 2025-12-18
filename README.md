
Conductor

Conductor is a headless, multi-tenant product feed distribution service.
It ingests normalized product data from clients, validates it against a canonical schema, and pushes delta-based updates to multiple third-party catalog channels (Google, Meta, Yotpo, etc.) using standardized, best-practice integrations.

The system is designed for high throughput, low operational cost, and clear separation of responsibilities between clients, the core platform, and channel-specific logic.

Core Design Goals

Multi-tenant SaaS architecture

Headless API (no UI assumptions)

Delta-based product updates (no full re-syncs by default)

Explicit lifecycle control per channel (active | inactive | delete)

No client-side SDKs required

High-volume batch ingestion (50,000+ products)

Low-cost, cloud-native operation

Enterprise-ready security and observability

High-Level Architecture

Conductor runs as two services:

API Service

Authenticates clients

Accepts product data

Validates and normalizes input

Tracks ingestion runs

Persists metadata and blobs

Publishes async jobs

Worker Service

Processes ingestion jobs

Computes deltas vs canonical state

Enqueues channel-specific push work

Pushes batches to third-party channels

Records results and errors

Services communicate asynchronously via a queue.

Cloud Platform

Google Cloud Platform (GCP)

Concern	Service
Compute	Cloud Run (2 services)
Database	Cloud SQL (MySQL)
Object Storage	Cloud Storage
Async Messaging	Pub/Sub
Secrets	Secret Manager / KMS
Logging & Metrics	Cloud Logging / Monitoring
Data Storage Strategy

Conductor uses a hybrid storage model:

Relational Database (Cloud SQL)

Stores:

Tenants

OAuth clients

Feeds

Channel subscriptions

Channel keys (metadata + encrypted credentials)

Runs and run summaries

Product indexes and hashes

Dirty queues

Push attempts

Error events

Object Storage (Cloud Storage)

Stores:

Raw ingestion payloads (NDJSON, gzip, short retention)

Normalized product snapshots (chunked, gzip, longer retention)

Lifecycle policies keep costs predictable.

Tenancy & Authentication

Each tenant authenticates via OAuth2 client credentials

One access token per tenant

All requests require a Bearer token

Tenant context is derived from the token

Third-party channel credentials are never exposed via the API.

Feeds & Channels
Feed

A Feed represents a tenant’s product distribution configuration.

One feed can push to multiple channels

Channels are enabled/disabled per feed

Each channel has standardized configuration

Channel Configuration (Customer-Controlled)

Per feed, per channel, customers may only control:

Whether pushing to that channel is enabled

Channel-specific keys (destination identifiers + credentials)

All connection logic, batching, retries, and best practices are standardized and owned by Conductor.

Product Model

Clients submit products using a canonical schema.

Key properties

product_key – unique product identifier

Required base fields (title, description, price, etc.)

Optional attributes (variants, options, metadata)

Per-channel control blocks

Per-Channel Lifecycle Control

Each product specifies intent per channel:

"channel": {
  "google": {
    "control": {
      "state": "active"
    }
  }
}


Valid states:

active – publish/update

inactive – archive/deactivate (channel-specific)

delete – delete from channel

Deletes are explicit only.

Ingestion Model
Supported Ingestion Methods

JSON upsert (small/medium batches)

NDJSON bulk upsert (large batches, streaming, gzip supported)

Ingestion Behavior

Products are validated and normalized

A stable hash is computed for each product

Products are compared to the last canonical state

Outcomes per product:

unchanged → no push

enqueued → scheduled for push

rejected → validation failed

If no products change, no push is triggered.

Delta-Based Push Model

No scheduled syncs

No inferred deletes

Pushes occur only when data changes or lifecycle state changes

Products are pushed in channel-appropriate batches

Channel workers operate independently

Runs, Status & Reporting

Every ingestion creates a Run.

Run tracking includes:

Counts (received, valid, rejected, unchanged, enqueued)

Push triggered or not

Per-product disposition

Per-channel push attempts

Detailed error events

APIs expose:

Run status

Error reports

Summary reports

Product-level inspection (optional)

Error Handling

Errors are captured at every stage:

Ingestion

Validation

Normalization

Channel push (request/response)

Errors are:

Structured

Persisted

Associated with run, product, channel, and stage

Unknown input fields are ignored but generate a once-per-run warning.

Security Model

OAuth2 for API access

Encrypted channel credentials at rest

Secrets never returned via API

Strict IAM separation between services

Auditability for config changes

Designed to exceed PCI-aligned security expectations

Design Principles

Explicit over implicit

Stable contracts over flexible templates

Channel logic isolated behind adapters

Cost-aware by default

Observability is a first-class concern

Descriptive, readable code over themed naming

Status

This repository currently contains:

Core service scaffolding

Configuration and logging foundations - incoming

Documented architecture and contracts - incoming

Channel implementations (Google, Meta, etc.) will be added incrementally once the core pipeline is complete.