# Kevinfoo Fullstack App

A modern full-stack web application built with Rust (Actix-web), MongoDB, HTMX, and Bulma.

## Prerequisites

- [Rust](https://www.rust-lang.org/tools/install) (latest stable)
- [MongoDB](https://www.mongodb.com/docs/manual/administration/install-community/) (running locally or a connection string)
- [cargo-tarpaulin](https://github.com/xd009642/tarpaulin) (for code coverage)

## Getting Started

1. **Clone the repository** (if not already in it).
2. **Configure Environment Variables**:
   ```bash
   cp .env.example .env
   ```
   Edit `.env` and set your `MONGODB_URI` (e.g., `mongodb://localhost:27017`).

3. **Install Dependencies and Run**:
   ```bash
   cargo run
   ```
   The application will be available at `http://localhost:8080`.

## Running Tests

To run the unit tests:
```bash
cargo test
```

## Code Coverage

We use `cargo-tarpaulin` for tracking code coverage. The configuration in `tarpaulin.toml` ensures that mock files and test utilities are excluded from the coverage score.

To generate a coverage report:
```bash
cargo tarpaulin
```

To generate an HTML report:
```bash
cargo tarpaulin --out Html
```

## Project Structure

- `src/main.rs`: Application entry point and server setup.
- `src/api/`: HTMX-friendly route handlers.
- `src/models/`: Data structures and MongoDB models.
- `src/repository/`: MongoDB data access layer.
- `src/services/`: Core business logic.
- `templates/`: Handlebars HTML templates.
- `static/`: Static assets (CSS/JS).
- `render.yaml`: Render.com blueprint for deployment.

## Deployment

This project is ready to be deployed on [Render.com](https://render.com) using the provided `render.yaml` blueprint. Ensure you set the `MONGODB_URI` environment variable in your Render dashboard.
