---
id: "003"
title: "Choose PostgreSQL as primary database"
status: "draft"
date: "2024-03-10"
---

# ADR-003: Choose PostgreSQL as primary database

## Status
Draft

## Context
We need to choose a primary relational database for the application. The main candidates are PostgreSQL and MySQL.

## Decision
Use PostgreSQL 16 as the primary relational database for its advanced features including JSONB support, full-text search, and strong concurrency handling.

## Consequences
Team needs PostgreSQL expertise. Excellent tooling and cloud provider support. JSONB allows flexible schema where needed. Slightly higher resource usage compared to MySQL for simple workloads.
