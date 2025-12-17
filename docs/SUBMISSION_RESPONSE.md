# ActInSpace 2026 - Submission Response

## Challenge #6: "Protect Airbus space assets from cyber threats"

### 1. Use of space data and technologies contextualized to the chosen challenge
**Score Target: 5/5**

This project is a direct response to the **ADS #6** challenge, implementing a "Secure-by-Design" and "Zero-Trust" architecture specifically tailored for space systems.

**Evidence of Contextualization:**
- **Space Protocols & Standards:** The platform explicitly models space-specific interactions:
    - **CCSDS-inspired Telemetry & Telecommand (TT&C):** The `ttc-gateway` and `satellite-sim` exchange structured commands (e.g., `deorbit`, `adjust-payload`) and telemetry frames, mimicking the Consultative Committee for Space Data Systems (CCSDS) packet standards.
    - **SPD-5 Compliance:** The architecture is built from the ground up to satisfy **U.S. Space Policy Directive-5** (SPD-5) principles, including:
        - Protection of command and control links (via the Zero-Trust Gateway).
        - Protection of ground systems (via the Space-SOC).
        - Supply chain security (via the DevSecOps plane).
- **Space Threat Modeling:** We use the **Aerospace Corporation's SPARTA** matrix and **ENISA's Space Threat Landscape** to define our threat scenarios. The `threat-library` contains executable scenarios that map directly to real-world space attacks (e.g., *T1005: Command Injection*, *T1008: Malicious Software Update*).
- **Satellite Lifecycle Support:** The platform covers the entire lifecycle, from software build (supply chain) to orbit operations (TT&C) and anomaly response (Space-SOC), reflecting the unique operational constraints of space missions (e.g., long-term maintenance via OTA updates).

### 2. Degree of innovation and originality of the product/service
**Classification: Both (Realistic Technological & Organisational Innovation)**
**Score Target: 5/5**

**Realistic Technological Innovation:**
- **Zero-Trust for Orbit:** Unlike traditional "perimeter-based" ground station security, this project brings **Zero-Trust Principles** primarily used in IT directly to the TT&C link. Every command is cryptographically verified and policy-checked against mission state *before* being uplinked.
- **Secure OTA for Space:** We implement a **TUF-inspired (The Update Framework)** secure Over-The-Air update mechanism for satellites. This addresses the critical vulnerability of "satellites functioning as legacy hardware in orbit" by enabling secure, signed patch management.

**Realistic Organisational Innovation:**
- **Space-SOC Integration:** We bridge the gap between Space Operations (Flight Dynamics/Mission Control) and Cyber Operations (CERT). The **Space-SOC** provides a unified view where telemetry anomalies (e.g., unexpected power draw) are correlated with cyber events (e.g., unauthorized login attempts), creating a holistic "Space Situational Awareness" for cyber threats.
- **Policy-as-Code for Mission Rules:** Using policy-as-code (OPA/Rego style) allows mission safety rules to be updated dynamically without recompiling flight software, increasing agility and safety.

### 3. Expected benefits of the project
**Classification: With the potential to impact a large number of lives globally / Solve major social issue**
**Score Target: 5/5**

**Impact on Critical Infrastructure:**
Satellites are effectively **critical infrastructure**. A cyberattack on a satellite constellation could disrupt:
- **Global Navigation (GPS/Galileo):** Affecting logistics, emergency response, and financial timing.
- **Earth Observation:** Impacting climate monitoring, disaster relief coordination, and national security.
- **Communication:** Cutting off remote areas and maritime/aviation connectivity.

**Value-Added Service:**
By preventing such attacks, the **Space Cyber Resilience Platform** ensures the **reliability and availability** of these essential services.
- **Employment:** The project supports the creation of new high-value roles: "Space Security Analysts" and "Space DevSecOps Engineers", bridging the skills gap between aerospace engineering and cybersecurity.
- **Sovereignty:** Enhances the strategic autonomy and resilience of national and commercial space assets against nation-state actors.

### 4. Relevance of the economic model
**Classification: Large and/or growing market with a clear and convincing business plan**
**Score Target: 5/5**

*(See detailed breakdown in `docs/BUSINESS_MODEL.md`)*

**Summary of Economic Viability:**
- **Market:** The global space economy is projected to reach **$1T by 2040** (Morgan Stanley). As fleets grow (Mega-constellations like Starlink, Kuiper), the attack surface expands exponentially, creating massive demand for automated security solutions.
- **Customers:**
    - **Mega-constellation Operators:** Need automated, scalable security (SOC) to manage thousands of assets.
    - **National Space Agencies & Defence:** Mandated to comply with strict new regulations (SPD-5, EU Space Law).
    - **Prime Contractors (e.g., Airbus):** Need to secure their supply chains and demonstrate compliance to varied customers.
- **Reliability:** The model is based on a mix of **Enterprise Licensing** (recurring SaaS/On-prem revenue) and **Cyber-Insurance Prevention** (reducing premiums for operators by demonstrating robust security posture).

---

## Team & Leadership
**Score Target: 5/5**

- **Structure:** Defined roles (DevSecOps Specialist, Space Systems Architect, Business Strategist).
- **Communication:** Proven ability to translate complex technical concepts (Zero-Trust) into business value (Risk Reduction).
- **Entrepreneurship:** Strategy focuses on "Dual-Use" technology (Commercial & Defence), maximizing funding opportunities and market reach.
