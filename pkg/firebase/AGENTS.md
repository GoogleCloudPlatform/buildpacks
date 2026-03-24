---
trigger:
  glob: "third_party/gcp_buildpacks/pkg/firebase/**"
---

# App Hosting Buildpacks - Agent Workspace Context

This directory contains the source code for Google Cloud Buildpacks logic
supporting **Firebase App Hosting**.

## 1. Overview & Purpose

App Hosting Buildpacks process user code and configurations during building and
deployment. They prepare environment variables, resolve secrets, and coalesce
multiple configuration layers into final deployment schemas.

--------------------------------------------------------------------------------

## 2. Technology Stack & Components

*   **Language**: Go (Standard libraries + schema structures).

### Key Packages

*   **`apphostingschema/`**: Validator methods parsing `apphosting.yaml`.
*   **`preparer/`**:
    *   Runs at **Build Time**.
    *   Validates and merges environment-specific YAML configurations.
    *   De-references keys and pins secret versions mapping to Secret Manager
        before writing `.env` maps to the CNB volume layers.
*   **`publisher/`**:
    *   Coalesces final build outputs.
    *   Merges `apphosting.yaml` and adapter-emitted `bundle.yaml` outputs into
        a structure suitable for updating App Hosting build requests.

--------------------------------------------------------------------------------

## 3. Coding Conventions & Guardrails

*   **Conflict Resolution**: When merging configurations in `publisher/` or
    `preparer/`, **DO** guarantee that user declarations from `apphosting.yaml`
    **always** win over framework-adapter bundle suggestions. Duplication logs
    are written during collision detections.
*   **Volume Mounts**: Pay close attention to output mappings writes (e.g.
    `/platform/env` format support for lifecycle commands vs pack). Do not break
    paths written without validating down-dependent reading tools support
    levels.

--------------------------------------------------------------------------------

## 4. Execution context & Injected Runners

To modify step injections cross-module, note that the step wrappers executing
`/bin/preparer` and `/bin/publisher` are **not arbitrary**. * **Location**:
`google3/cloud/serverless/boq/runtime/config/buildspec/fahgenerator.go` *
**Logic**: The **Serverless RCS** backend compiles the Cloud Build `.Build`
struct forming steps (`preparerStep`, `packStep`, `publisherStep`). * **Usage**:
Do not rename volume mounts (e.g. `/yaml/apphostingyaml_processed` or
`/output_bundle`) inside Buildpacks without also updating the corresponding flag
strings in `fahgenerator.go` concurrently inside your changelist to avoid
breaking rollout cycles!
