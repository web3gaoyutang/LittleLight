from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


CHECKS = {
    "docs/openapi.yaml": [
        "openapi: 3.",
        "/api/v1/dashboard:",
        "Course:",
    ],
    "app/.env.example": [
        "VITE_API_BASE_URL=/api/v1",
        "VITE_DEV_API_TARGET=http://localhost:8080",
    ],
    ".env.example": [
        "APP_ENV=",
        "HTTP_ADDR=",
        "DATABASE_URL=",
        "REDIS_ADDR=",
        "REDIS_DB=",
        "MIGRATIONS_DIR=",
        "AI_PROVIDER=",
        "LLM_API_KEY=",
        "LLM_BASE_URL=",
        "LLM_MODEL=",
    ],
    "deploy/nginx/default.conf": [
        "location /readyz",
    ],
    "deploy/docker-compose.yml": [
        "condition: service_healthy",
        "http://localhost/healthz",
    ],
    "docs/deployment-runbook.md": [
        "pg_dump",
        "FLUSHDB",
        "down migration",
        "docker-build",
        "readyz",
        "日志与排障",
        "docker compose -f deploy/docker-compose.yml logs -f api",
        "PostgreSQL unavailable",
        "Redis unavailable",
        "LLM parent drafts failed",
        "H5 cannot reach API",
    ],
    "docs/database-schema.md": [
        "users",
        "courses",
        "parent_profiles",
        "reminders",
        "communication_records",
        "healing_entries",
        "ai_generations",
        "favorites",
        "idx_courses_user_weekday",
        "002_seed.sql",
    ],
    "docs/engineering-verification-matrix.md": [
        "uni-app",
        "Golang",
        "PostgreSQL",
        "Redis",
        "Docker",
        "verify-all.ps1",
        "docker-build",
        "scripts/verify-docker.ps1",
    ],
}


def main() -> None:
    for relative_path, required_fragments in CHECKS.items():
        path = ROOT / relative_path
        if not path.exists():
            raise SystemExit(f"{relative_path} is missing")
        text = path.read_text(encoding="utf-8")
        for fragment in required_fragments:
            if fragment not in text:
                raise SystemExit(f"{relative_path} is missing required fragment: {fragment}")
    print("documentation coverage ok")


if __name__ == "__main__":
    main()
