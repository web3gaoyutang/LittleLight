from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[1]

STRICT_REQUEST_SCHEMAS = [
    "WechatLoginInput",
    "UserProfileInput",
    "NotificationSettingsInput",
    "FavoriteInput",
    "CourseInput",
    "ReminderInput",
    "ParentProfileInput",
    "CommunicationRecordInput",
    "HealingEntryInput",
    "AIActionInput",
    "ParentDraftRequest",
    "PraiseRequest",
]


CHECKS = {
    "docs/openapi.yaml": [
        "openapi: 3.",
        "/api/v1/dashboard:",
        "Course:",
    ],
    "app/.env.example": [
        "VITE_API_BASE_URL=/api/v1",
        "VITE_APP_API_BASE_URL=https://api.example.com/api/v1",
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
        "XF_ASR_APP_ID=",
        "XF_ASR_API_KEY=",
        "XF_ASR_API_SECRET=",
    ],
    "README.md": [
        "uni-app + Vue 3",
        "Golang HTTP API",
        "PostgreSQL",
        "Redis",
        "Docker Compose",
        "scripts\\verify-all.ps1",
        "scripts/api-smoke.mjs",
        "npm run test:h5-smoke",
        "integration",
        "docs/engineering-technical-design.md",
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
        "scripts/api-smoke.mjs",
        "npm run test:h5-smoke",
        "integration",
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
    openapi = yaml.safe_load((ROOT / "docs/openapi.yaml").read_text(encoding="utf-8"))
    if not str(openapi.get("openapi", "")).startswith("3."):
        raise SystemExit("docs/openapi.yaml is not an OpenAPI 3 document")
    required_paths = ["/api/v1/dashboard", "/api/v1/auth/wechat", "/api/v1/me/favorites"]
    for path in required_paths:
        if path not in openapi.get("paths", {}):
            raise SystemExit(f"docs/openapi.yaml is missing path: {path}")
    schemas = openapi.get("components", {}).get("schemas", {})
    for schema in ["Course", "AIDraft", "PageInfo"]:
        if schema not in schemas:
            raise SystemExit(f"docs/openapi.yaml is missing schema: {schema}")
    for schema_name in STRICT_REQUEST_SCHEMAS:
        schema = schemas.get(schema_name)
        if not schema:
            raise SystemExit(f"docs/openapi.yaml is missing schema: {schema_name}")
        if schema.get("additionalProperties") is not False:
            raise SystemExit(f"docs/openapi.yaml schema {schema_name} must set additionalProperties: false")
    snooze_schema = (
        openapi.get("paths", {})
        .get("/api/v1/reminders/{id}/snooze", {})
        .get("post", {})
        .get("requestBody", {})
        .get("content", {})
        .get("application/json", {})
        .get("schema", {})
    )
    if snooze_schema.get("additionalProperties") is not False:
        raise SystemExit("docs/openapi.yaml snooze request body must set additionalProperties: false")
    print("documentation coverage ok")


if __name__ == "__main__":
    main()
