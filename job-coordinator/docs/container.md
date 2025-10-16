# CQL SDK Container Architecture

This document describes the containerization strategy for the CQL SDK and how it integrates with various services in the system.

## CQL Container Design

The following diagram illustrates the four-layer architecture, showing how services interact with containerized CQL SDK entry points.

```mermaid
block-beta
  columns 3

  %% LAYER 4 — Callers/Services
  svc_exec["☁️ Service: CQL Job Runner<br/>Batch/stream orchestration; calls Execute"]
  svc_build["🎨 Service: UI / Build Task<br/>User-driven pipeline; produces versioned libraries"]
  svc_pop["📊 Service: Population Build Process<br/>ETL/Population prep; triggers FHIR validation"]

  space:3

  %% LAYER 3 — Entry Points (same image, different commands)
  exec["▶️ Entry: Execute CQL<br/>Run measures/rules over patient bundles"]
  buildlib["🔨 Entry: Build Library<br/>Compile & validate CQL→ELM; package artifacts"]
  validate["✅ Entry: Validate FHIR<br/>FHIR schema/profile validation & conformance checks"]

  %% LAYER 2 — Containerization
  container["🐳 Linux Container<br/>Single, versioned image packaging the CQL SDK and CLIs"]:3

  %% LAYER 1 — Core SDK
  core["⚙️ .NET CQL SDK<br/>Core evaluation engine, parsers, temporal ops, model types"]:3

  %% FLOWS BETWEEN LAYERS
  svc_exec --> exec
  svc_build --> buildlib
  svc_pop --> validate

  %% STYLING
  classDef services fill:#e3f2fd,stroke:#1565c0,stroke-width:2px
  classDef entries fill:#fff3e0,stroke:#e65100,stroke-width:2px
  classDef container fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px
  classDef sdk fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px

  class svc_exec,svc_build,svc_pop services
  class exec,buildlib,validate entries
  class container container
  class core sdk
```

**Layer 1 - Core SDK**: The .NET CQL SDK provides the foundational evaluation engine, parsers, temporal operations, and model types.

**Layer 2 - Containerization**: A single, versioned Linux container image packages the CQL SDK and CLI tools, ensuring consistent runtime environments.

**Layer 3 - Entry Points**: The same container image supports multiple entry points (execute, build, validate) through different commands.

**Layer 4 - Services**: Various services invoke the appropriate entry points based on their needs (job execution, library building, FHIR validation).

## Measure Build and Execution Pipeline

This workflow shows the end-to-end process from measure authoring/import through build, caching, and parallel execution across pods.

### Key Components

**Input Sources**:

- Custom CQL files authored by users
- HEDIS measures imported from external sources (C# and DLLs are stripped to ensure we build with our versioned SDK)

**Base Measure Library**: Central repository storing CQL files and FHIR measure bundles (metadata).

**SDK Version Selection**: Each measure run configuration specifies which SDK version to use for building, ensuring compatibility and reproducibility.

**Parallel Builds**: Multiple measures can be built simultaneously using different SDK versions as needed.

**Filestore Cache**: Versioned measure bundles are stored in a shared filestore, organized by SDK version, which execution pods mount as volumes.

**Highly Parallel Execution**: Multiple pods execute different measures concurrently, each using the appropriate SDK version specified during the build phase.

```mermaid
flowchart TB
    %% INPUT SOURCES
    A1[Author Custom CQL] --> ML
    A2[Import HEDIS Measures] --> Strip[Strip C# & DLLs]
    Strip --> ML

    %% BASE MEASURE LIBRARY
    ML[("Base Measure Library<br/>CQL Files + FHIR Measure Bundles")]

    %% MEASURE RUN CONFIGURATION
    ML --> RunConfig[Measure Run Config]
    SDK_Select[Select/Default SDK Version] --> RunConfig

    %% TRIGGER BUILD
    RunConfig --> Trigger{Trigger Build}

    %% PARALLEL MEASURE BUILDS
    Trigger --> Build1[Build Measure 1<br/>with SDK vX]
    Trigger --> Build2[Build Measure 2<br/>with SDK vX]
    Trigger --> Build3[Build Measure N<br/>with SDK vY]

    %% SAVE TO FILESTORE CACHE
    Build1 --> Cache
    Build2 --> Cache
    Build3 --> Cache
    Cache[("Filestore Cache<br/>Versioned Measure Bundles<br/>by SDK version")]

    %% EXECUTION QUEUE
    Cache --> Queue[Measure Execution Queue]

    %% EXECUTION PODS
    subgraph Pods[Execution Pods - Highly Parallel]
        direction LR
        Pod1["Pod 1<br/>Executing Measure 1<br/>on SDK vX"]
        Pod2["Pod 2<br/>Executing Measure 2<br/>on SDK vX"]
        Pod3["Pod 3<br/>Executing Measure N<br/>on SDK vY"]
        PodN["Pod ...<br/>Executing Measure ...<br/>on SDK ..."]
    end

    %% MOUNT VOLUMES
    Queue --> Pods
    Cache -.mount volumes.-> Pods

    %% RESULTS
    Pods --> Results[Execution Results]

    %% STYLING
    classDef library fill:#e1f5ff,stroke:#01579b,stroke-width:2px
    classDef build fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef cache fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef pod fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px

    class ML,Cache library
    class Build1,Build2,Build3 build
    class Cache cache
    class Pod1,Pod2,Pod3,PodN pod
```
