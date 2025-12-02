
<!-- plan.md -->

# Space Cyber Resilience Platform – Development Plan

This plan assumes a **single primary developer** working part-time. Timelines are indicative and can be compressed or extended depending on availability.

---

## Phase 0 – Project bootstrap (≈ 1 week)

**Goals**

- Establish repository structure and basic tooling.
- Document the vision in a way that is understandable for:
  - Innovation contest juries (e.g. ActInSpace)
  - MSc supervisors / academic reviewers
  - Potential collaborators

**Tasks**

- [ ] Initialise repository with:
  - `README.md`
  - `ARCHITECTURE.md`
  - `plan.md`
- [ ] Create base directory structure:
  - `infra/`, `supply-chain/`, `ttc-gateway/`, `space-soc/`, `satellite-sim/`, `ground-station-sim/`, `threat-library/`, `docs/`
- [ ] Add basic `.editorconfig`, `.gitignore`, and Go / Node project boilerplate.
- [ ] Set up minimal CI (linting, basic build checks).

**Deliverables**

- Public or private Git repository with core documentation and skeleton structure.

---

## Phase 1 – Minimal end-to-end MVP (≈ 2–3 weeks)

**High-level objective**

Implement a **thin vertical slice** that runs from:

> Developer → CI → signed build → satellite-sim → TT&C gateway → Space-SOC dashboard

### 1.1 Supply-chain MVP

- [ ] Implement a simple Go service for `satellite-sim` (no complex logic yet).
- [ ] Add CI job to build container image for `satellite-sim`.
- [ ] Integrate at least one SAST and one SCA step into the pipeline.
- [ ] Generate a basic SBOM (e.g. CycloneDX via open-source tool).
- [ ] Implement a minimal signing script / service:
  - Input: image digest / artefact file
  - Output: signed metadata file (JSON)

### 1.2 TT&C Gateway MVP

- [ ] Implement `ttc-gateway` as a Go service with:
  - Single `/command` HTTP endpoint.
  - Simple token-based authentication.
  - Hard-coded allow/deny policy for 1–2 command types.
- [ ] Implement `ground-station-sim` CLI / small API to send commands.
- [ ] Connect `ttc-gateway` to `satellite-sim` via an internal HTTP / gRPC call.
- [ ] Add structured logging for all incoming commands and decisions.

### 1.3 Space-SOC MVP

- [ ] Implement `space-soc/backend` with:
  - Simple log ingestion endpoint (HTTP).
  - SQLite / PostgreSQL schema for command events.
- [ ] Implement `space-soc/frontend` with:
  - Table / timeline view of command events (who, what, when, decision).
- [ ] Wire up:
  - `ttc-gateway` → Space-SOC ingestion
  - CI pipeline → Space-SOC ingestion (e.g. on successful build)

### 1.4 Infra

- [ ] Create `infra/docker-compose.yaml` that runs:
  - `satellite-sim`
  - `ttc-gateway`
  - `space-soc` backend + frontend
  - Optional: DB

**Exit criteria**

- You can:
  - Run `docker compose up`.
  - Open the Space-SOC UI.
  - Send a command via `ground-station-sim` and see it appear in the dashboard.

---

## Phase 2 – Threat modelling & Zero-Trust hardening (≈ 3–4 weeks)

**High-level objective**

Add real **space-cyber flavour** by:

- Introducing basic **Zero-Trust** controls for TT&C.
- Building an initial **threat library** aligned with ENISA / SPARTA / SPACE-SHIELD references. :contentReference[oaicite:22]{index=22}  

### 2.1 Policy-as-code for TT&C

- [ ] Integrate a simple policy engine (OPA / custom DSL) into `ttc-gateway`.
- [ ] Define a small policy set:
  - Different roles (operator, engineer, admin).
  - Safe vs risky commands (e.g. payload toggle vs orbit change).
  - Mission phase constraints.
- [ ] Expose policy decisions (reason codes) in logs and Space-SOC.

### 2.2 Anomaly detection (rule-based)

- [ ] Implement basic in-memory / DB-backed anomaly detection:
  - Rate thresholds per command type.
  - Time-of-day anomalies.
- [ ] Send anomaly events to Space-SOC and display them clearly.

### 2.3 Threat library (v1)

- [ ] Create `threat-library/` with 3–5 canonical scenarios:
  - Malicious OTA update attempt.
  - Unauthorised operator sending dangerous commands.
  - Uplink spoofing / flood (simulated).
- [ ] For each scenario, document:
  - Description, objectives, assumed attacker.
  - Mapping to SPARTA / SPACE-SHIELD tactics where relevant.
  - Expected observables (logs, anomalies).
- [ ] Implement small scripts or CLIs to **replay** each scenario in the lab.

### 2.4 Space-SOC enhancements

- [ ] Add views linking events to threat scenarios:
  - Per-scenario timeline.
  - “Re-run scenario” button that triggers replay scripts.
- [ ] Add simple incident object:
  - Title, severity, related events.

**Exit criteria**

- You can run a recorded scenario (e.g. “Malicious OTA update”) and:
  - See relevant events and anomalies.
  - View the mapped scenario and incident in the Space-SOC UI.

---

## Phase 3 – Supply-chain resilience & OTA (≈ 3–4 weeks)

**High-level objective**

Strengthen the **supply-chain plane** to reflect current best practices in space system cybersecurity. :contentReference[oaicite:23]{index=23}  

### 3.1 OTA workflow

- [ ] Implement an OTA client in `satellite-sim`:
  - Polls registry / controller for new versions.
  - Downloads artefact + attestation.
  - Verifies signature and SBOM policy.
- [ ] Implement OTA controller (in `supply-chain/` or separate service):
  - Authorises updates based on mission policy (e.g. freeze during critical windows).
  - Records all update attempts and results.

### 3.2 SBOM and policy checks

- [ ] Parse SBOM artifacts to enforce simple policies (e.g. disallow packages with known CVEs).
- [ ] Surface SBOM / risk information in Space-SOC:
  - Show which satellites are running vulnerable components.

### 3.3 DevSecOps reporting

- [ ] Add a CI report ingestion into Space-SOC:
  - Build history per component.
  - Recent security scan results.
- [ ] Display a “software posture” panel per satellite / ground component.

**Exit criteria**

- OTA updates follow a clearly defined, secure flow.
- Malicious or non-compliant updates are blocked and visible in Space-SOC.
- You can show a “software posture” view to a reviewer / jury.

---

## Phase 4 – Contest / MSc / publication packaging (≈ 2–4 weeks)

This phase focuses on turning the technical prototype into **submission-ready artefacts**.

### 4.1 ActInSpace / competition packaging

- [ ] Create a short **pitch deck** (10–12 slides) summarising:
  - Problem and threat landscape (using ENISA / SPD-5 references). :contentReference[oaicite:24]{index=24}  
  - Solution architecture and key innovations.
  - Market / user segments (satellite operators, defence, insurance, regulators).
- [ ] Prepare a 3–5 minute **demo recording**:
  - Start from Space-SOC overview.
  - Show normal operations.
  - Trigger at least one attack scenario and show detection / mitigation.
- [ ] Write a concise **one-pager** summarising the value proposition.

### 4.2 Academic / MSc alignment

- [ ] Draft a 2–4 page **technical white paper**:
  - Context: space-cyber threat landscape.
  - Design: three planes and key design principles.
  - Evaluation: what scenarios are covered; limitations.
- [ ] Map the project onto potential MSc modules / thesis topics:
  - Space cybersecurity.
  - Secure DevSecOps for critical infrastructure.
  - Threat modelling for hybrid (IT / space) systems.

---

## Phase 5 – Optional extensions

Depending on time and interest:

- **Advanced analytics**
  - Incorporate basic ML techniques for anomaly detection on telemetry and command sequences.
- **More realistic simulation**
  - Add latency / packet loss models and multi-node constellations.
- **Integration with external tools**
  - Export events to third-party SIEM / SOAR.
  - Integrate with cyber ranges or CTF frameworks.

---

## Progress tracking

A simple approach is to:

- Use GitHub Projects / Issues to track each task.
- Tag issues by plane:
  - `plane:devsecops`
  - `plane:ttc-gateway`
  - `plane:soc`
- Maintain a short changelog in `CHANGELOG.md` once the project becomes active.

---

## Success criteria

The project can be considered “successful” for a first major milestone when:

1. A reviewer can:
   - Start the stack with a single command.
   - Send commands and see them in the Space-SOC.
   - Trigger at least one documented attack scenario and observe clear, comprehensible defences.

2. The documentation (README + ARCHITECTURE + threat model) is:
   - Understandable without live explanation.
   - Sufficiently grounded in current space-cyber references.

3. The project is **credible** as:
   - An answer to **ActInSpace ADS #6**.
   - A flagship side project for cybersecurity / space-security oriented MSc applications.
   - A foundation for future competitions or publications.
