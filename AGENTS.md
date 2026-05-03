# Vibe Coding Agent Guidelines

This document outlines the architecture, tech stack, and coding standards for our application. All AI agents must adhere to these guidelines when generating or modifying code for this project.

## 1. Backend (Rust & Actix)

- **Framework**: Rust using the `actix-web` framework.
- **Folder Structure**: Follow Rust best practices for structuring a web application (e.g., separating concerns):
  - `src/main.rs`: Entry point and server initialization.
  - `src/api/` or `src/handlers/`: Route handlers and HTTP-specific logic.
  - `src/models/`: Data structures, database models, and request/response DTOs.
  - `src/repository/`: Database interactions and queries.
  - `src/services/`: Core business logic.
  - `src/utils/` or `src/common/`: Shared helpers, error types, and configurations.
- **Testing & Code Coverage**:
  - Implement comprehensive unit tests for all components.
  - Ensure high code coverage.
  - **CRITICAL**: Any mock files, fixtures, or test utilities created for unit testing MUST NOT count toward the code coverage score. Agents must configure the coverage tool (e.g., via `tarpaulin.toml` or `#[cfg(not(tarpaulin_include))]` macros) to explicitly ignore mock and test-only files.

## 2. Frontend (HTMX & Bulma)

- **Core Tech**: Server-side rendered HTML heavily utilizing **HTMX** for dynamic, SPA-like interactions without writing custom JavaScript.
- **Styling**: Use the **Bulma CSS** framework for all styling and layout.
- **Dependencies constraint**: Strictly limit the use of 3rd party components and external JavaScript libraries. Use as few external dependencies as absolutely possible, relying instead on HTMX and Bulma defaults.

## 3. Database (MongoDB)

- **Primary Datastore**: **MongoDB**.
- Utilize the official `mongodb` crate for Rust. 
- Follow NoSQL data modeling best practices, keeping documents properly structured and leveraging indexes.

## 4. Hosting & Deployment (Render.com)

- **Platform**: The entire application will be hosted on **Render.com**.
- **Blueprint Configuration**: A `render.yaml` Blueprint file is REQUIRED at the root of the repository to define the infrastructure as code. Ensure all environment variables, build commands, start commands, and service definitions (including database connection strings) are correctly specified in this blueprint.
