
---

```markdown
<!-- ARCHITECTURE.md -->

# Space Cyber Resilience Platform – Architecture

This document describes the architecture of the **Space Cyber Resilience Platform**, designed as a concrete response to **ActInSpace ADS #6 – “Protect Airbus space assets from cyber threats”**.

The platform models a simplified but realistic **space system** and applies contemporary **space-cybersecurity guidance** from ENISA, SPD-5 and space-specific threat frameworks such as SPARTA / SPACE-SHIELD. :contentReference[oaicite:11]{index=11}  

---

## 1. Scope and assumptions

- **Scope**
  - Single mission with:
    - One or more **satellite-sim** nodes
    - One **ground-station-sim** (Mission Control)
    - A shared **DevSecOps / CI** environment
    - A central **Space-SOC**
  - Focus is on **cybersecurity** aspects (identity, integrity, visibility, resilience), not on orbital mechanics or precise RF physics.

- **Assumptions**
  - All components run in containers on commodity hardware.
  - Network characteristics are approximated (latency, packet loss can be simulated later).
  - No real mission data or proprietary information is used.

---

## 2. Design principles

The architecture is guided by:

1. **Risk-based and lifecycle-wide security** – from SPD-5 (design, development, deployment, operations, decommissioning). :contentReference[oaicite:12]{index=12}  
2. **Zero-Trust Architecture (ZTA)** – identity-centric, least-privilege, continuous verification on all links (especially TT&C). :contentReference[oaicite:13]{index=13}  
3. **Threat-informed defence** – using ENISA’s space threat analysis and SPARTA / SPACE-SHIELD style matrices to model TTPs and detection logic. :contentReference[oaicite:14]{index=14}  
4. **Secure DevSecOps and supply chain** – secure builds, SBOMs, attestation, and controlled updates to space assets. :contentReference[oaicite:15]{index=15}  
5. **Composability and observability** – each plane is independently deployable, but all emit structured telemetry to the Space-SOC.

---

## 3. Logical architecture overview

```mermaid
flowchart LR
    subgraph DevSecOps & Supply Chain
        DEV[Developer]
        CI[CI/CD Pipeline]
        SIGN[Signing & Attestation]
        REG[Artefact Registry]
    end

    subgraph Control & Communications (TT&C)
        GS[Ground-Station-Sim]
        GATEWAY[TT&C Zero-Trust Gateway]
        SAT[Satellite-Sim]
    end

    subgraph Operations & Intelligence (Space-SOC)
        INGEST[Ingestion & Normalisation]
        CORR[Correlation & Analytics]
        DASH[Analyst Dashboard]
        TLIB[Threat Scenario Library]
    end

    DEV --> CI --> SIGN --> REG
    REG -->|Secure OTA| SAT

    GS -->|Authenticated Command| GATEWAY --> SAT
    SAT -->|Telemetry| GATEWAY

    GATEWAY -->|Logs & Metrics| INGEST
    SAT -->|Telemetry & Events| INGEST
    CI -->|Build & Release Events| INGEST

    INGEST --> CORR --> DASH
    TLIB --> CORR
```
4. Component breakdown
4.1 DevSecOps & Supply Chain (Plane 1)

Objectives

Ensure that software destined for space assets is built, scanned, signed and traced end-to-end.

Model supply-chain guidance from SPD-5 and ENISA (supply-chain attacks, malicious updates, dependency risk). 
stli.iii.org.tw
+2
ResearchGate
+2

Key components

supply-chain/ci/ – CI/CD pipelines

Build container images / binaries for satellite-sim, ground-station-sim, ttc-gateway, etc.

Run SAST and SCA.

Generate SBOMs.

supply-chain/signing-service/

Signs artefacts and generates attestation metadata (build ID, time, test results, scan status).

Exposes verification API for other components.

supply-chain/registry/ (logical; can be a standard registry)

Stores signed images / packages.

Provides metadata to OTA and Space-SOC.

OTA orchestrator (part of supply-chain/ and satellite-sim/)

Secure update protocol for satellite-sim:

Requests new version + attestation

Verifies signature and policy

Applies update or rejects with reason

4.2 TT&C Zero-Trust Gateway (Plane 2)

Objectives

Enforce Zero-Trust principles on the most critical interface: commands to space assets.

Provide a single control point for policy enforcement, telemetry and audit.

Key components

ttc-gateway/api/

Receives commands from authenticated clients (ground-station-sim, potential automation).

Normalises and validates commands.

ttc-gateway/policy-engine/

Evaluates each command against:

Operator identity & role

Satellite state (e.g. normal / safe mode)

Mission rules and risk thresholds

Implemented with policy-as-code (e.g. REGO / DSL).

ttc-gateway/anomaly-detector/

Tracks command patterns (frequency, type, timing).

Raises alerts on anomalies (e.g. burst of destructive commands).

ttc-gateway/audit-logger/

Records full details of every decision (allow/deny) and forwards to Space-SOC.

satellite-sim/

Represents one or more spacecraft:

Receives validated commands from the gateway.

Emits telemetry, state changes and events.

ground-station-sim/

Emulates operator console / automation:

Authenticates to gateway.

Sends mission commands.

Renders basic status and responses.

4.3 Operations & Intelligence – Space-SOC (Plane 3)

Objectives

Provide situational awareness, attack detection and training / simulation capabilities.

Integrate space-specific threat knowledge (SPARTA / SPACE-SHIELD, ENISA Space Threat Landscape). 
Space & Cybersecurity Info
+3
enisa.europa.eu
+3
aerospace.org
+3

Key components

space-soc/backend/

Ingestion layer

Collects logs / events from:

TT&C gateway

Satellite and ground-station simulators

Supply-chain pipelines

Normalises into a common schema (e.g. JSON events).

Storage layer

Time-series / columnar store for events.

Relational DB for assets, scenarios, and incidents.

Correlation & analytics

Rules / queries mapping event patterns to known TTPs (SPARTA / ATT&CK style).

Basic incident creation / scoring.

space-soc/frontend/

Mission overview:

Assets, satellites, ground segments, their software versions and risk posture.

Attack timelines:

Commands, anomalies, telemetry anomalies, CI events.

Threat scenario explorer:

Visual representation of attack flows across planes.

threat-library/

JSON / YAML definitions of attack scenarios:

Example: “Malicious OTA update”, “Uplink spoofing attempt”, “Ground IT compromise pivoting to TT&C”.

Each scenario references:

Tactics / techniques (SPARTA / SPACE-SHIELD IDs where applicable).

Expected observables (log features).

Playbook steps.

5. Data flows
5.1 Normal software release and update

Developer pushes changes to satellite-sim code.

CI pipeline builds artefact, runs SAST/SCA, generates SBOM.

Signing service signs artefact and emits attestation.

Artefact is pushed to registry.

OTA orchestrator on satellite-sim:

Requests latest artefact + attestation.

Verifies signature and policies.

Applies update and emits events.

All steps generate events ingested by Space-SOC.

5.2 Normal TT&C command

Operator authenticates to ground-station-sim.

Operator issues a command (e.g. “adjust payload mode”).

ground-station-sim sends a signed, authenticated request to TT&C gateway.

Policy engine validates:

Identity / role

Command vs satellite state / mission phase

Gateway forwards command to satellite-sim and records full audit log.

Satellite executes and returns telemetry; all relevant events go to Space-SOC.

5.3 Example attack: malicious OTA update

Attacker attempts to inject a modified image into registry or bypass CI.

When satellite-sim requests an update:

OTA orchestrator pulls artefact and attestation.

Signature or attestation fails, policy denies the update.

Space-SOC correlates:

CI logs (unexpected artefact)

Registry anomalies

OTA denial events

A high-severity incident is created and shown to analysts.

5.4 Example attack: uplink hijack / unauthorised commands (simulated)

Attacker node sends raw commands directly to TT&C gateway or satellite-sim.

Gateway rejects unauthenticated traffic; anomaly detector flags unusual patterns.

Satellite-sim either never receives commands, or rejects them at its own security layer.

Space-SOC shows an attack timeline with source, attempted commands and defence mechanisms.

6. Threat model (summary)
6.1 Assets

Space segment

Satellite-sim software and configuration

Onboard update mechanism and keys

Ground segment

Ground-station-sim application

TT&C gateway

DevSecOps / CI environment

Shared infrastructure

Artefact registry

Logs and telemetry datastore

Identity provider (conceptual)

6.2 Trust boundaries

Between CI / signing service and registry

Between ground-station-sim and TT&C gateway

Between TT&C gateway and satellite-sim

Between all components and Space-SOC

6.3 Representative threats

T1 – Malicious or compromised software updates

Supply-chain / CI compromise; malicious images deployed to satellites.

T2 – Unauthorised command injection

Stolen credentials, weak auth, direct uplink spoofing attempts.

T3 – Telemetry and log tampering

Attackers suppress or alter evidence to evade detection.

T4 – Multi-segment campaigns

Ground IT compromise pivoting into satellite operations.

6.4 Mitigations mapping (high level)

T1 – Use signed artefacts, attestation, SBOM and policy checks; align with SPD-5 and ENISA supply-chain guidance. 
stli.iii.org.tw
+2
ResearchGate
+2

T2 – Enforce identity-aware TT&C gateway, least-privilege policies, command signing, anomaly detection; aligned with Zero-Trust for space communications. 
cyber
+2
aerospace.org
+2

T3 – Centralised, append-only logging, integrity checks on logs, and independent monitoring in Space-SOC.

T4 – Holistic visibility where CI, gateway and satellite telemetry all feed correlation and scenario detection based on SPARTA / SPACE-SHIELD style attack flows. 
Industrial Cyber
+3
aerospace.org
+3
Space & Cybersecurity Info
+3

A more detailed threat model and mapping to specific SPARTA / SPACE-SHIELD techniques can be captured in docs/THREAT_MODEL_SPARTA_ENISA.md.

7. Non-functional considerations

Modularity – each plane can be deployed and tested independently.

Extensibility

Additional satellite types or missions can be added by instantiating more satellite-sim nodes.

New attack scenarios can be added to threat-library/ without changing core code.

Performance

Initial implementation targets a lab environment; real-time constraints are not strict.

Design should allow later simulation of latency / bandwidth constraints.

Security

Even in a lab, secrets handling, least privilege, and hardening should be applied where practical.

Portability

Pure containerised deployment for local development; small K3s cluster for multi-node testing.

8. Future extensions

Integrate with actual cyber ranges (e.g. Airbus-style CyberRange concepts) for training. 
Airbus

Implement more advanced analytics (ML-based anomaly detection on telemetry and commands).

Introduce more detailed RF / orbital simulation for realistic timing and topology.

Expose external APIs so that third-party tools can plug into the Space-SOC.
