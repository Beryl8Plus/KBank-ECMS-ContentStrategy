# ECMS API Specification Project Mandates

This document defines the core mandates, technical standards, and procedural knowledge for the KBank ECMS Rule Management API project. All agents working on this project MUST adhere to these instructions.

## 1. Project Context
- **Project Name:** KBank ECMS (Enterprise Content Management System)
- **Primary Domain:** Rule Management API
- **Repository:** `KBank-ECMS-Backend`
- **Key Endpoint:** `POST /rule-management` (Active API endpoint)

## 2. Technical Standards

### 2.1 Database & Schema (Single Source of Truth)
- **Primary Schema Reference:** `docs/diagram/models.md`
- **Mandate:** Any changes to the API models or database interactions MUST be validated against the schema defined in `models.md`.
- **Naming Conventions:** Follow the snake_case naming convention as defined in the schema (e.g., `decision_rules`, `login_token_histories`).

### 2.2 Documentation & Localization
- **Data Dictionary:** `docs/diagram/data_dictionary_th.md` contains the authoritative Thai language descriptions for all database fields.
- **Mandate:** All user-facing documentation, field descriptions, and business-level explanations MUST use the Thai terminology defined in the Data Dictionary.

### 2.3 Framework: BMad
- This project utilizes the **BMad (Building Modular Agentic Designs)** framework.
- Core configuration and module-specific settings are located in `_bmad/`.
- **Initialization:** Use the `bmad-init` skill to resolve configuration variables before executing module-specific tasks.

## 3. Specialized Workflows

### 3.1 Rule Management Development
1. **Research:** Map the requested rule change to the tables in `models.md`.
2. **Implementation:** Update Go structs in `internal/model/` (or create new ones) to match the schema.
3. **Verification:** Create a reproduction script or curl command (see `README.md`) to verify the endpoint behavior.

### 3.2 Skill Creation & Maintenance
- When adding new project-specific capabilities, use the `skill-creator` to initialize a new skill in `.gemini/skills/`.
- Ensure each new skill includes:
  - `SKILL.md` with YAML frontmatter.
  - Project-specific references (e.g., linking to `models.md`).

## 4. Operational Guardrails
- **No Data Fabrication:** Never invent database fields or API parameters that are not defined in the schema.
- **Security First:** Protect the `.sfdx` and `_bmad/_config` directories from being logged or committed.
- **Thai Language First:** When describing data fields to Thai stakeholders, always provide the Thai description from the Data Dictionary.
