# Dez Cron Scheduler

A fast, dynamic HTTP Cron Job scheduling Backend and Headless API Dashboard built with **Go (Gin framework)**, **MongoDB**, and **Robfig/cron/v3**.

## ✨ Features

- **Dynamic Scheduling**: Create, update, or remove cron schedules dynamically via API without ever restarting the server.
- **Timezone Aware**: Execute jobs based on region, supporting custom timezones (`Asia/Jakarta`) per job.
- **Resilience & Alerting**: Supports dynamic `retry_count`, `disabled_after` consecutive execution failures, and sends failure reports to Discord/Slack (by providing an `alert_webhook_url` per scheduled job).
- **Actionable Logs**: Every execution payload response is saved automatically into a `job_logs` Table for your analysis. Memory-safe: truncates response bodies to 20KB max and relies on MongoDB's TTL Native Caching to auto-delete logs older than 2 weeks!
- **Zero-Dependency Headless Admin UI**: Manage authentication entirely locally. A protected API using unlimited generated multiple revokable `Bearer` tokens! Access the generation panel on `GET /` login screen with credentials from `.env`.

## ⚙️ Prerequisites

- Go 1.21+
- MongoDB Database

## 🚀 Application Setup

1. **Copy the example environment variables** and configure your DB details:

    ```bash
    cp .env.example .env
    ```

2. **Run the server locally**:

    ```bash
    go mod tidy
    go run main.go
    ```

3. **Log in to Admin Dashboard**:
   Open a browser to `http://localhost:8080/` and sign in with default credentials `admin:admin` (Unless you changed it on `.env`). Generate a Token to test the API securely or fetch Data.

4. **API Endpoints List Documentation**:
   The native OpenAPI (Redoc) reference is available securely at: `http://localhost:8080/docs`

## 🐳 Hosting & Docker

This repository includes a multi-stage `Dockerfile` and is fully ready to be deployed on Railway, Heroku, or VPS Cloud services statically!
You can run it cleanly as a container:

```bash
docker build -t dez-cron .
docker run -p 8080:8080 --env-file .env dez-cron
```
