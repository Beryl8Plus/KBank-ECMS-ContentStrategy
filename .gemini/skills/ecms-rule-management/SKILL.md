---
name: ecms-rule-management
description: Manage and document KBank ECMS decision rules, schemas, and API specifications. Use when working with decision_rules, schedules, placements, and attributes in the KBank-ECMS-Backend repository.
---

# ECMS Rule Management

This skill provides specialized workflows and references for working on the KBank ECMS Rule Management API.

## Core Mandates

1. **Schema Integrity:** Always refer to `docs/diagram/models.md` for the single source of truth for the database schema.
2. **Localization:** Use `docs/diagram/data_dictionary_th.md` for all Thai language descriptions and business terms.
3. **BMad Framework:** This project uses BMad. Always initialize with the `bmad-init` skill to resolve configuration variables.

## Workflows

### Mapping Rule Requirements
When a new rule requirement is provided:
1. Identify the relevant tables in `models.md`.
2. Use the Thai descriptions from `data_dictionary_th.md` to confirm business logic with stakeholders.
3. Verify that the requested field types match the schema.

### API Specification Updates
When updating the API spec:
1. Ensure the `POST /rule-management` endpoint is updated to reflect any model changes.
2. Update the `schema_registry` table entry if the frontend UI payload changes.

## References

- **Models:** [docs/diagram/models.md](../../../docs/diagram/models.md)
- **Thai Data Dictionary:** [docs/diagram/data_dictionary_th.md](../../../docs/diagram/data_dictionary_th.md)
- **Project README:** [README.md](../../../README.md)

## Guardrails
- **No Manual Commits:** Never commit changes unless explicitly requested.
- **Data Consistency:** All new attributes MUST be registered in the `attributes` table before being used in `rule_condition`.
