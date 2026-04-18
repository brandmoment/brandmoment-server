# Task: JWT Auth Flow — Request to DB
Profile: Research
Stage: Done
Created: 2026-04-18
Updated: 2026-04-18 11:22

## Context
Исследование: как работает JWT авторизация от HTTP запроса до БД-запроса.

## Report
Final report compiled as `03-report.md`. Covers:
- Complete JWT flow from HTTP request through middleware, handler, service, repository to SQL
- Two access patterns: top-level organizations (JWT membership check at handler) vs sub-resources (SQL org_id filter)
- 15 security findings: 4 CRITICAL, 3 HIGH, 5 MEDIUM, 3 LOW
- Positive findings: architectural correctness, sqlc usage, proper error handling
