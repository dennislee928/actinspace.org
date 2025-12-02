# Demo Script - Space Cyber Resilience Platform

## Duration: 3-5 minutes

## Setup (Before Demo)

1. Ensure all services are running:
   ```bash
   docker compose -f infra/docker-compose.yaml up -d
   docker compose -f infra/docker-compose.yaml ps
   ```

2. Open browser tabs:
   - Tab 1: http://localhost:3001 (Space-SOC Dashboard - Events)
   - Tab 2: http://localhost:3001/posture (Software Posture)
   - Tab 3: Terminal ready for commands

3. Clear any old events (optional):
   ```bash
   docker compose -f infra/docker-compose.yaml down -v
   docker compose -f infra/docker-compose.yaml up -d
   ```

## Demo Flow

### Part 1: Introduction (30 seconds)

**Script**:
> "Welcome to the Space Cyber Resilience Platform—a comprehensive cybersecurity solution for space systems. This platform protects satellites across their entire lifecycle: from secure software development, through zero-trust command and control, to unified security operations."

**Action**: Show architecture diagram (Slide 5 from pitch deck or `ARCHITECTURE.md`)

---

### Part 2: Normal Operations (30 seconds)

**Script**:
> "Let's start with normal operations. Here's our Space-SOC dashboard showing real-time events from all system components."

**Action**: 
1. Show Space-SOC Dashboard (Tab 1)
2. Point out the event timeline
3. Send a normal command:
   ```bash
   go run ground-station-sim/cmd/ground-station-sim/main.go \
     -gateway http://localhost:8081 \
     -cmd health_check \
     -token operator-token
   ```
4. Show the event appearing in the dashboard

---

### Part 3: Attack Scenario 1 - Unauthorized Command (60 seconds)

**Script**:
> "Now, let's simulate an attack. An operator with insufficient privileges attempts to send a dangerous 'deorbit' command that could destroy the satellite."

**Action**:
1. Send unauthorized command:
   ```bash
   go run ground-station-sim/cmd/ground-station-sim/main.go \
     -gateway http://localhost:8081 \
     -cmd deorbit \
     -token operator-token
   ```

2. Show terminal output (command denied)

3. Switch to Space-SOC Dashboard:
   - Point out the policy_decision event (denied)
   - Show the reason: "requires admin role"
   - Highlight the severity level

**Script**:
> "The Zero-Trust gateway immediately blocked this command based on policy. The operator role is not authorized for dangerous commands. This event is logged with full context for audit and investigation."

---

### Part 4: Anomaly Detection (45 seconds)

**Script**:
> "The platform also detects anomalous behavior patterns. Let's trigger an anomaly by sending commands rapidly."

**Action**:
1. Send burst of commands:
   ```bash
   for i in {1..12}; do
     go run ground-station-sim/cmd/ground-station-sim/main.go \
       -gateway http://localhost:8081 \
       -cmd test_cmd \
       -token operator-token &
   done
   wait
   ```

2. Refresh Space-SOC Dashboard

3. Point out anomaly_detected events:
   - Command burst detected
   - Time-of-day anomalies (if outside normal hours)
   - Rate limit violations

**Script**:
> "Notice the anomaly detection events. The system flagged this unusual burst of commands—15 commands in 10 seconds exceeds our threshold. These anomalies help security teams identify potential attacks even when individual commands appear legitimate."

---

### Part 5: Incidents & Correlation (30 seconds)

**Script**:
> "High-severity events automatically create security incidents for investigation."

**Action**:
1. Click "安全事件" (Incidents) tab
2. Show auto-created incidents
3. Point out:
   - Severity levels
   - Related events
   - Status tracking

**Script**:
> "The Space-SOC automatically correlates related events into incidents, providing analysts with a clear picture of potential attacks and their scope."

---

### Part 6: Software Posture (30 seconds)

**Script**:
> "The platform also tracks the security posture of all software components."

**Action**:
1. Navigate to Software Posture page (Tab 2)
2. Show component cards:
   - Current versions
   - Vulnerability counts
   - Update availability
   - Last scan times

**Script**:
> "This Software Posture dashboard shows which versions are running, known vulnerabilities, and available updates. It's fed by our secure OTA update system and SBOM analysis."

---

### Part 7: Supply Chain Protection (45 seconds)

**Script**:
> "Let's demonstrate the supply chain protection. I'll register a new software version."

**Action**:
1. Register a release:
   ```bash
   curl -X POST http://localhost:8084/api/v1/releases \
     -H "Content-Type: application/json" \
     -d '{
       "component": "satellite-sim",
       "version": "v1.1.0",
       "imageDigest": "sha256:demo123",
       "attestation": "{\"digest\":\"sha256:demo123\",\"signature\":\"demo_sig\"}"
     }'
   ```

2. Show response (status: pending)

3. Approve the release:
   ```bash
   curl -X POST http://localhost:8084/api/v1/releases/1/approve
   ```

4. Check satellite logs:
   ```bash
   docker compose -f infra/docker-compose.yaml logs satellite-sim | tail -10
   ```

**Script**:
> "New software versions require approval before distribution. The satellite's OTA client checks for updates every 30 seconds, verifies signatures, and applies approved updates. This prevents malicious updates from reaching space assets."

---

### Part 8: Threat Library (30 seconds)

**Script**:
> "For training and red-team exercises, we've defined 5 canonical threat scenarios aligned with SPARTA and ENISA frameworks."

**Action**:
1. Show `threat-library/scenarios/` directory
2. Open one scenario YAML file
3. Point out:
   - SPARTA tactic mappings
   - Expected observables
   - Playbook steps

**Script**:
> "These scenarios can be replayed using our CLI tool, allowing security teams to practice detection and response in a safe environment."

---

### Part 9: Wrap-Up (20 seconds)

**Script**:
> "In summary, the Space Cyber Resilience Platform provides:
> - **Integrated protection** across supply chain, operations, and incident response
> - **Space-aware security** that understands mission context
> - **Proven capabilities** with working implementation
> - **Standards alignment** with SPD-5, ENISA, and SPARTA
> 
> This is not just a concept—it's a working platform ready for pilot deployments. Thank you!"

**Action**: Return to architecture diagram or summary slide

---

## Backup Demos (If Time Permits)

### Replay Threat Scenario

```bash
go run threat-library/scripts/replay-scenario.go \
  -scenario threat-library/scenarios/unauthorized-dangerous-command.yaml
```

### SBOM Policy Check

```bash
go run supply-chain/sbom/cmd/check-sbom/main.go \
  -sbom supply-chain/sbom/examples/satellite-sim-v1.0.0.cdx.json
```

## Troubleshooting During Demo

### Services Not Responding
```bash
docker compose -f infra/docker-compose.yaml restart
```

### Dashboard Not Loading
- Check browser console for errors
- Verify backend is healthy: `curl http://localhost:8083/health`

### Commands Failing
- Verify TT&C Gateway is running: `curl http://localhost:8081/health`
- Check token format: Should be `Bearer operator-token` or `Bearer admin-token`

## Q&A Preparation

**Q: Is this production-ready?**
A: This is a working prototype demonstrating the architecture and key capabilities. Production deployment would require hardening, scalability improvements, and integration with real satellite systems.

**Q: How does this compare to existing solutions?**
A: Unlike generic IT security tools, this platform is space-aware, integrates supply chain through operations, and uses space-specific threat intelligence (SPARTA/SPACE-SHIELD).

**Q: What about performance?**
A: Current implementation targets lab/training environments. Production deployment would optimize for latency, throughput, and high availability.

**Q: Can this integrate with existing systems?**
A: Yes, the modular architecture allows integration with existing ground systems, CI/CD pipelines, and security tools via standard APIs.

**Q: What about regulatory compliance?**
A: The platform implements SPD-5 principles and ENISA recommendations, providing audit trails and policy enforcement needed for compliance.

