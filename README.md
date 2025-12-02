<!-- README.md -->

# Space Cyber Resilience Platform

> End-to-end cyber protection for space systems — from secure software supply chain, to zero-trust command and control, to space-aware security operations.

This repository implements a **prototype platform** for the ActInSpace challenge:

> **ADS #6 – “Protect Airbus space assets from cyber threats”**

The goal is to show how a modern **Zero-Trust, threat-informed, DevSecOps** approach can protect a space system across its full lifecycle:

- **Build & supply chain** – secure development and deployment of space software
- **Control & communications** – zero-trust protection of **TT&C** (telemetry, tracking and command)
- **Operations & incident response** – **Space-SOC / Space-CERT** style monitoring, simulation and forensics

The design aligns with current guidance such as:

- **ENISA Space Threat Landscape** (commercial satellites under increasing cyber attack, including jamming, hijacking and network exploitation)
- **U.S. Space Policy Directive-5 (SPD-5)** – cybersecurity principles for space systems (risk-based design, supply-chain security, protection of command links, incident response)
- **Space-specific threat frameworks** such as **SPARTA / SPACE-SHIELD**, which extend MITRE-style TTPs to space systems

This is a **simulation-only** environment: it does not connect to any real satellite or operational ground system.

---

## 1. High-level concept

The platform is organised into three planes:

1. **Supply-Chain Plane**  
   Secure DevSecOps and software supply chain for space workloads:
   - CI pipeline with SAST / SCA scanning
   - SBOM generation and artefact attestation
   - Image / firmware signing and policy checks
   - Simulated secure OTA (over-the-air) update workflow to “satellite nodes”

2. **Control & Communications Plane (TT&C Zero-Trust Gateway)**  
   A **Zero-Trust TT&C Gateway** in front of all commands to the satellite:
   - Strong identity and authorisation for operators
   - Policy-as-code for command approval (least privilege, mission rules)
   - Command signing and integrity checks
   - Basic behaviour anomaly detection (e.g. abnormal frequency / timing)
   - Full audit logging for forensics

3. **Operations & Intelligence Plane (Space-SOC / Space-CERT)**  
   A **Space-aware SOC** that:
   - Ingests logs and telemetry from the TT&C gateway, simulated satellites, and CI pipeline
   - Maps events to **space-specific attack flows** (SPARTA / ATT&CK-style)
   - Provides dashboards, timelines and basic correlation
   - Hosts a small **threat scenario library** for training and red-team / blue-team exercises

For detailed diagrams and data flows, see [`ARCHITECTURE.md`](./ARCHITECTURE.md).

---

## 2. Repository layout (proposed)

```text
.
├── README.md
├── ARCHITECTURE.md
├── plan.md
├── docs/
│   ├── THREAT_MODEL_SPARTA_ENISA.md
│   ├── USE_CASES.md
│   └── COMPLIANCE_SPD5_ENISA.md
├── infra/
│   ├── docker-compose.yaml
│   └── k8s-manifests/
├── supply-chain/             # Plane 1 – DevSecOps & supply chain
│   ├── ci/
│   │   └── .gitlab-ci.yml
│   ├── sbom/
│   └── signing-service/
├── ttc-gateway/              # Plane 2 – Zero-trust TT&C gateway
│   ├── cmd/
│   └── internal/
├── space-soc/                # Plane 3 – Space-SOC backend + UI
│   ├── backend/
│   └── frontend/
├── satellite-sim/            # Simulated spacecraft node
├── ground-station-sim/       # Simulated MCC / operator console
└── threat-library/           # SPARTA / ENISA-aligned scenarios
```

## 3. Core features

### 3.1 Space DevSecOps & supply chain (Plane 1)

- Containerised build pipelines for space workloads (satellite and ground software)
- Static analysis (SAST) and dependency / image scanning (SCA) integrated into CI
- SBOM generation (e.g. CycloneDX / SPDX) for all deployable artefacts
- Artefact signing and attestation (build metadata, test results, scan status)
- Simulated secure OTA update flow from ground to satellite-sim nodes
- Traceability from production nodes back to artefacts and source commits

This plane is intended to demonstrate how SPD-5 and related guidance on supply-chain security can be operationalised for space systems.

### 3.2 Zero-Trust TT&C Gateway (Plane 2)

- Single logical gateway for all commands towards space assets
- Identity-aware access control (e.g. JWT / OIDC tokens or mTLS)
- Policy-as-code engine to validate each command against:
  - Operator role
  - Mission phase / satellite state
  - Risk level of the command
- Command signing / verification and integrity checks
- Rate limiting and baseline-based anomaly detection for TT&C traffic
- Rich audit trail for all command decisions (allow / deny / override)

### 3.3 Space-SOC / Space-CERT (Plane 3)

- **Centralised ingestion of:**
  - TT&C gateway logs (commands, decisions, anomalies)
  - Satellite and ground-station telemetry
  - CI / supply-chain events (builds, scans, releases, updates)
- Mapping of observed events to space-specific attack flows using open references such as SPARTA / SPACE-SHIELD and ENISA's space threat landscape.
- **Analyst dashboards:**
  - Mission overview and asset inventory
  - Attack timeline and affected assets
  - Scenario "replays" for training / exercises
- Hooks for future integration with cyber ranges and red-team tooling

## 4. Example scenarios

The platform is designed to support several canonical scenarios:

- **Malicious or compromised operator**
  - An operator attempts to send dangerous commands (e.g. de-orbit, disable payload).
  - Zero-Trust TT&C policies block or challenge the command; Space-SOC raises alerts and records evidence.

- **Malicious software update**
  - A build artefact with known vulnerabilities or altered origin attempts to be deployed as a satellite update.
  - The supply-chain plane rejects it (failed scans, invalid attestation); the update is never accepted by satellite-sim.

- **Uplink hijack / spoofed commands (simulated)**
  - An attacker node sends raw commands that bypass the official ground-station-sim.
  - TT&C gateway discards unauthenticated traffic; anomalous patterns are highlighted in the SOC.

- **Coordinated campaign across ground and space segments**
  - By correlating CI events, TT&C logs and telemetry, Space-SOC reconstructs a multi-stage attack that began in the ground IT environment and propagated towards space assets.

These are documented in more detail in `docs/USE_CASES.md` (to be implemented).

## 5. Technology stack

**Languages**

- Go (backend services: TT&C gateway, SOC API, signing service, simulators)
- TypeScript / React (Space-SOC frontend, operator UI)

**Infrastructure**

- Docker & Docker Compose (local lab)
- Optionally Kubernetes / K3s for multi-node simulation
- PostgreSQL or SQLite (event store and SOC data)
- Message broker (e.g. NATS / Redis Streams) for telemetry and command bus

**Security & DevSecOps**

- Static analysis and SCA tools (e.g. Semgrep / Trivy) integrated into CI
- SBOM generation tools (CycloneDX / Syft or equivalent)
- Simple in-repo signing / verification (e.g. based on Go crypto; pluggable to cosign / in-toto patterns later)

## 6. Quickstart

**Prerequisites**

- Docker & Docker Compose
- Go ≥ 1.23
- Node.js ≥ 20

**Clone repository**

```bash
git clone https://github.com/<your-account>/space-cyber-resilience-platform.git
cd space-cyber-resilience-platform
```

**Start the lab environment**

```bash
# 啟動所有服務
docker compose -f infra/docker-compose.yaml up -d

# 查看服務狀態
docker compose -f infra/docker-compose.yaml ps

# 查看日誌
docker compose -f infra/docker-compose.yaml logs -f
```

**Access the dashboards**

- **Space-SOC Dashboard**: http://localhost:3001
  - 事件列表和時間軸
  - 安全事件 (Incidents)
  - 軟體姿態面板: http://localhost:3001/posture

**Test the system**

1. **Send commands via ground-station-sim**:
   ```bash
   # Build the CLI
   go build -o ground-station-sim ./ground-station-sim/cmd/ground-station-sim
   
   # Send a command (will be denied - operator role)
   ./ground-station-sim -gateway http://localhost:8081 -cmd deorbit -token operator-token
   
   # Send with admin role (will be allowed)
   ./ground-station-sim -gateway http://localhost:8081 -cmd deorbit -token admin-token
   ```

2. **Replay threat scenarios**:
   ```bash
   # Build replay tool
   go build -o replay-scenario ./threat-library/scripts/replay-scenario.go
   
   # Replay unauthorized command scenario
   ./replay-scenario -scenario threat-library/scenarios/unauthorized-dangerous-command.yaml
   ```

3. **Test OTA workflow**:
   ```bash
   # Run Phase 3 test script
   pwsh scripts/test-phase3.ps1
   ```

**View results in Space-SOC**

- Navigate to http://localhost:3001
- View events, anomalies, and policy decisions
- Check Incidents tab for security events
- View Software Posture for component versions and vulnerabilities

## 7. Implementation Status

The development roadmap and milestones are tracked in [`plan.md`](.cursor/plan/plan.md).

### ✅ Phase 0 - Project Bootstrap (Completed)
- Repository structure and documentation
- Basic CI/CD configuration
- Development tooling setup

### ✅ Phase 1 - Minimal End-to-End MVP (Completed)
- Satellite-sim service with command interface
- TT&C Gateway with token-based auth and basic policy
- Space-SOC backend (event ingestion, storage)
- Space-SOC frontend (event timeline)
- Docker Compose orchestration

### ✅ Phase 2 - Threat Modelling & Zero-Trust (Completed)
- Policy-as-code engine with 4 default rules
- Rule-based anomaly detection (rate limit, time-of-day, burst, role activity)
- 5 threat scenarios defined (YAML format)
- Scenario replay CLI tool
- Incidents API and management
- Enhanced Space-SOC UI with severity and anomaly display

### ✅ Phase 3 - Supply-Chain Resilience & OTA (Completed)
- OTA Controller with approval workflow
- OTA Client in satellite-sim (periodic update checks)
- SBOM parser and policy checker
- Software Posture tracking and dashboard
- Mission phase-aware update control
- Signature verification workflow

### 🔄 Phase 4 - Contest/Academic Packaging (In Progress)
- Documentation updates
- Demo scripts and guides

### 📋 Phase 5 - Extensions (Planned)
- ML-based anomaly detection
- More realistic simulation (latency, packet loss)
- External SIEM/SOAR integration

## 8. Disclaimer

This repository is for research and educational purposes only.

All satellites, ground systems and attacks are simulated.

The project does not represent Airbus, CNES, ENISA, NIST or any space operator.

## 9. Acknowledgements & references (non-exhaustive)

- **ENISA** – Space Threat Landscape reports
- **U.S. Space Policy Directive-5** – Cybersecurity Principles for Space Systems
- **Aerospace Corporation** – SPARTA Matrix for space-cyber TTPs
- **SPACE-SHIELD** and related space cyber frameworks
- **Airbus** – Cyber Programmes / Defence and Space cyber activities

