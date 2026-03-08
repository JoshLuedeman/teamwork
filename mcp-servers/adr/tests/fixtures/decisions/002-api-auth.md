---
id: "002"
title: "Use JWT for API authentication"
status: "accepted"
date: "2024-02-01"
---

# ADR-002: Use JWT for API authentication

## Status
Accepted

## Context
We need a stateless authentication mechanism for our REST API that works across microservices.

## Decision
Use JSON Web Tokens (JWT) with RS256 signing for API authentication. Tokens will be issued by a central auth service and verified by each microservice independently.

## Consequences
Stateless authentication reduces database lookups. Token expiry must be managed carefully. Need secure key management for RS256 signing keys.
