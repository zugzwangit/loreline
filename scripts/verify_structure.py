from __future__ import annotations

import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    ".openai/hosting.json",
    "api/openapi.yaml",
    "app/features/management/Management.tsx",
    "app/features/operations/Operations.tsx",
    "app/features/workspace/Workspace.tsx",
    "app/lib/api.ts",
    "app/lib/domain.ts",
    "deploy/kubernetes.yaml",
    "docs/repository-layout.md",
    "pnpm-lock.yaml",
    "pnpm-workspace.yaml",
    "services/gateway/README.md",
    "services/gateway/go.sum",
    "services/intelligence/README.md",
    "services/intelligence/requirements.lock",
    "tooling/sites-vite-plugin.ts",
}
FORBIDDEN_PREFIXES = (
    ".next/",
    ".vinext/",
    ".wrangler/",
    "dist/",
    "examples/",
    "node_modules/",
    "work/",
)
FORBIDDEN_FILES = {
    "dev.err.log",
    "dev.log",
    "public/file.svg",
    "public/globe.svg",
    "public/window.svg",
}


def tracked_files() -> set[str]:
    result = subprocess.run(
        ["git", "ls-files"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return {line.strip().replace("\\", "/") for line in result.stdout.splitlines() if line.strip()}


def main() -> None:
    tracked = tracked_files()
    missing = sorted(path for path in REQUIRED if not (ROOT / path).is_file())
    forbidden = sorted(
        path
        for path in tracked
        if (ROOT / path).exists() and (path in FORBIDDEN_FILES or path.startswith(FORBIDDEN_PREFIXES))
    )
    if missing or forbidden:
        details = []
        if missing:
            details.append("missing required files: " + ", ".join(missing))
        if forbidden:
            details.append("generated or starter files are tracked: " + ", ".join(forbidden))
        raise SystemExit("; ".join(details))
    print(f"Repository structure verified ({len(tracked)} tracked files).")


if __name__ == "__main__":
    main()
