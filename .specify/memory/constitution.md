<!--
SYNC IMPACT REPORT
==================
Version Change: NONE → 1.0.0
Type: MAJOR (initial constitution)

Modified Principles:
- [NEW] I. Demo Often
- [NEW] II. Test Always
- [NEW] III. Reuse - Don't Reinvent
- [NEW] IV. Reduce Iteration Time
- [NEW] V. User/Dev Experience is Key

Added Sections:
- Core Principles (all 5 principles)
- Technology Stack
- Quality Standards
- Governance

Templates Requiring Updates:
- ✅ .specify/templates/plan-template.md (to be validated)
- ✅ .specify/templates/spec-template.md (to be validated)
- ✅ .specify/templates/tasks-template.md (to be validated)
- ✅ .specify/templates/agent-file-template.md (to be validated)

Follow-up TODOs:
- None
-->

# DQME Constitution

## Core Principles

### I. Demo Often

**Principle**: Functionality MUST be demonstrable at every meaningful increment.

- Every feature, service, or component MUST reach a demonstrable state within one development cycle
- Demonstrations MUST occur in environments that mirror production constraints (GCP resources, FHIR data structures, CQL execution contexts)
- Stakeholder feedback loops MUST be short: weekly demos preferred, never exceed two-week intervals
- "Demo-ready" means: deployable, testable via realistic scenarios, with visible outputs

**Rationale**: Healthcare quality measures impact real clinical decisions. Early and frequent
validation with stakeholders prevents costly rework and ensures the platform meets actual clinical
workflow needs. Demonstrable progress maintains alignment across infrastructure, services, and
container teams.

### II. Test Always (NON-NEGOTIABLE)

**Principle**: All code MUST have automated tests before merge.

- Unit tests MUST cover all business logic, CQL parsing, FHIR bundle processing, and measure calculation functions
- Integration tests MUST validate: service-to-service contracts, GCP resource interactions (Cloud Run, Pub/Sub, Firestore), end-to-end measure execution pipelines
- Test data MUST include representative FHIR bundles and CQL measure definitions
- CI pipelines MUST enforce: all tests pass, minimum code coverage thresholds (80%+ for core logic), no test skips without documented justification
- Tests MUST be written BEFORE implementation (TDD) for new features; bug fixes MUST include regression tests

**Rationale**: Clinical quality measure calculations must be accurate and auditable. Automated
testing is the only scalable way to ensure correctness across the complex interactions between CQL
engines, FHIR resources, and distributed cloud services. Test failures block merges.

### III. Reuse - Don't Reinvent

**Principle**: Prefer existing proven solutions over custom implementations.

- MUST evaluate existing libraries, GCP services, and open-source tools before building custom solutions
- For CQL execution: use established engines (e.g., cql-engine, fhir-cql) rather than custom parsers
- For FHIR handling: use HAPI FHIR or Google FHIR SDK rather than manual JSON parsing
- Infrastructure as Code: prefer GCP-native constructs (Cloud Run, Cloud Build) over custom orchestration
- Custom code is justified ONLY when: no existing solution meets requirements, integration cost exceeds build cost, or performance/security mandates it
- Document the evaluation process for any "build vs. buy" decision in architecture decision records (ADRs)

**Rationale**: Healthcare interoperability standards (FHIR, CQL) are complex and well-supported by
mature ecosystems. Reinventing these components introduces unnecessary risk, delays delivery, and
creates maintenance burden. Focus engineering effort on platform-specific value, not commodity
infrastructure.

### IV. Reduce Iteration Time

**Principle**: Minimize feedback cycles across development, testing, and deployment.

- Local development environments MUST support: hot-reload for code changes, mocked GCP services (Firestore emulator, Pub/Sub emulator), sample FHIR bundles and CQL measures for rapid testing
- Build times MUST be optimized: container image layer caching, incremental builds, parallel test execution
- Deployment pipelines MUST complete in under 15 minutes for dev/staging environments
- Code reviews MUST start within 4 business hours; approvals within 24 hours for non-breaking changes
- Terraform/infrastructure changes MUST have plan previews before apply
- Eliminate wait states: automate approvals where safe, parallelize independent tasks, provide immediate feedback on failures

**Rationale**: Long iteration cycles compound across a distributed team working on infrastructure,
services, and containers. Fast feedback enables rapid experimentation, quicker bug fixes, and
maintains developer momentum. In a healthcare platform, faster iterations mean faster delivery of
quality measure capabilities to end users.

### V. User/Dev Experience is Key

**Principle**: Both end-user interfaces and developer tooling MUST prioritize usability.

**For End Users (Clinicians, Quality Analysts)**:

- Measure execution results MUST be clear, actionable, and auditable (show which patients met criteria, why)
- Error messages MUST be human-readable and suggest corrective actions
- APIs MUST provide consistent, well-documented interfaces (OpenAPI specs required)
- Performance MUST meet expectations: measure execution under 30 seconds for typical patient populations

**For Developers**:

- Onboarding MUST be documented: setup scripts, README with prerequisites, sample data provided
- Local dev environment setup MUST complete in under 30 minutes
- Error messages in logs MUST include context: request IDs, trace spans, relevant resource identifiers
- Documentation MUST be co-located with code: inline comments for complex logic, architecture diagrams for service interactions
- Deployment processes MUST be scripted and repeatable (no manual GCP console clicks)

**Rationale**: Poor user experience (clinical or developer) leads to adoption failure. Clinicians
will reject a quality measure platform that produces confusing results. Developers will struggle
with a platform that has obscure errors and complex setup. Investing in experience pays dividends
in velocity, reliability, and user satisfaction.

## Technology Stack

**Cloud Platform**: Google Cloud Platform (GCP)

- Compute: Cloud Run for containerized services
- Storage: Firestore for state, Cloud Storage for measure definitions and results
- Messaging: Pub/Sub for event-driven workflows
- Infrastructure: Terraform for IaC

**Standards**:

- FHIR R4 for patient data representation
- CQL 1.5+ for quality measure logic
- HL7 quality measure specifications (FHIR Measure resource)

**Languages & Frameworks**: (To be defined per service; prefer consistency where possible)

- Container base images MUST use Google's distroless or slim variants
- Dependency management MUST be explicit (lock files required)

## Quality Standards

**Code Reviews**:

- All changes MUST be reviewed by at least one team member
- Reviews MUST verify: constitution compliance, test coverage, documentation updates

**Performance**:

- Measure execution latency MUST be monitored and alerted (p95 < 30s target)
- GCP resource usage MUST be tracked (cost per measure execution)

**Security**:

- All GCP services MUST use IAM least-privilege principles
- Secrets MUST be stored in Secret Manager, never in code or environment variables
- FHIR data MUST be handled per HIPAA requirements (encryption at rest and in transit)

**Observability**:

- All services MUST emit structured logs (JSON format) with trace context
- Distributed tracing MUST be enabled (Cloud Trace)
- Key metrics MUST be exported to Cloud Monitoring (request rates, error rates, latencies)

## Governance

**Authority**: This constitution supersedes all other development practices and policies. In cases of conflict, constitution principles take precedence.

**Amendment Process**:

1. Proposed amendments MUST be documented with rationale and impact analysis
2. Amendments require approval from project technical leads
3. Version numbering follows semantic versioning:
   - MAJOR: Breaking changes to principles or governance (e.g., removing a principle)
   - MINOR: New principles or sections added
   - PATCH: Clarifications, typo fixes, non-semantic updates
4. All amendments MUST include a migration plan if existing code/practices are affected

**Compliance**:

- All pull requests MUST be validated against constitution principles during code review
- Deviations require explicit justification and documented exceptions
- Regular constitution reviews (quarterly) to ensure relevance and effectiveness

**Version**: 1.0.0 | **Ratified**: 2025-10-02 | **Last Amended**: 2025-10-02
