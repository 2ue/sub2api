#!/usr/bin/env python3

from __future__ import annotations

import argparse
import csv
import json
import re
import sys
from dataclasses import dataclass
from datetime import date
from pathlib import Path

ROUTE_FILES = [
    "backend/internal/server/routes/common.go",
    "backend/internal/server/routes/auth.go",
    "backend/internal/server/routes/user.go",
    "backend/internal/server/routes/sora_client.go",
    "backend/internal/server/routes/gateway.go",
    "backend/internal/server/routes/admin.go",
    "backend/internal/setup/handler.go",
]

AUTH_USER_FILES = {
    "backend/internal/server/routes/auth.go",
    "backend/internal/server/routes/user.go",
    "backend/internal/server/routes/sora_client.go",
}

SECTION_ORDER = {
    "common": 0,
    "auth-user": 1,
    "gateway": 2,
    "admin-core": 3,
    "admin-accounts": 4,
    "admin-ops": 5,
    "admin-misc": 6,
}

ADMIN_CORE_PREFIXES = (
    "/api/v1/admin/api-keys",
    "/api/v1/admin/dashboard",
    "/api/v1/admin/groups",
    "/api/v1/admin/settings",
    "/api/v1/admin/subscriptions",
    "/api/v1/admin/users",
)

ADMIN_ACCOUNTS_PREFIXES = (
    "/api/v1/admin/accounts",
    "/api/v1/admin/antigravity",
    "/api/v1/admin/gemini",
    "/api/v1/admin/openai",
    "/api/v1/admin/proxies",
    "/api/v1/admin/sora",
)

GROUP_ASSIGN_RE = re.compile(r"(?P<var>[A-Za-z_]\w*)\s*:=\s*(?P<base>[A-Za-z_]\w*)\.Group\(")
ROUTE_CALL_RE = re.compile(r"(?P<var>[A-Za-z_]\w*)\.(?P<method>GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|Any)\(")


@dataclass
class Route:
    method: str
    path: str
    handler: str
    file: str
    line: int
    section: str


def remove_line_comment(line: str) -> str:
    result: list[str] = []
    in_string = False
    in_raw = False
    escape = False

    for index, char in enumerate(line):
        next_char = line[index + 1] if index + 1 < len(line) else ""

        if in_raw:
            result.append(char)
            if char == "`":
                in_raw = False
            continue

        if in_string:
            result.append(char)
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == '"':
                in_string = False
            continue

        if char == "`":
            in_raw = True
            result.append(char)
            continue

        if char == '"':
            in_string = True
            result.append(char)
            continue

        if char == "/" and next_char == "/":
            break

        result.append(char)

    return "".join(result)


def paren_delta(text: str) -> int:
    delta = 0
    in_string = False
    in_raw = False
    escape = False

    for char in text:
        if in_raw:
            if char == "`":
                in_raw = False
            continue

        if in_string:
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == '"':
                in_string = False
            continue

        if char == "`":
            in_raw = True
            continue
        if char == '"':
            in_string = True
            continue
        if char == "(":
            delta += 1
        elif char == ")":
            delta -= 1

    return delta


def join_paths(base: str, suffix: str) -> str:
    if not base:
        return suffix or ""
    if not suffix:
        return base
    if suffix.startswith("/"):
        return base.rstrip("/") + suffix
    return base.rstrip("/") + "/" + suffix.lstrip("/")


def extract_call_args(statement: str, open_index: int) -> str:
    depth = 0
    in_string = False
    in_raw = False
    escape = False

    for index in range(open_index, len(statement)):
        char = statement[index]

        if in_raw:
            if char == "`":
                in_raw = False
            continue

        if in_string:
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == '"':
                in_string = False
            continue

        if char == "`":
            in_raw = True
            continue
        if char == '"':
            in_string = True
            continue

        if char == "(":
            depth += 1
            continue
        if char == ")":
            depth -= 1
            if depth == 0:
                return statement[open_index + 1 : index]

    raise ValueError("unterminated call")


def split_top_level_args(args_text: str) -> list[str]:
    parts: list[str] = []
    current: list[str] = []
    depth_paren = 0
    depth_brace = 0
    depth_bracket = 0
    in_string = False
    in_raw = False
    escape = False

    for char in args_text:
        if in_raw:
            current.append(char)
            if char == "`":
                in_raw = False
            continue

        if in_string:
            current.append(char)
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == '"':
                in_string = False
            continue

        if char == "`":
            in_raw = True
            current.append(char)
            continue
        if char == '"':
            in_string = True
            current.append(char)
            continue

        if char == "(":
            depth_paren += 1
            current.append(char)
            continue
        if char == ")":
            depth_paren -= 1
            current.append(char)
            continue
        if char == "{":
            depth_brace += 1
            current.append(char)
            continue
        if char == "}":
            depth_brace -= 1
            current.append(char)
            continue
        if char == "[":
            depth_bracket += 1
            current.append(char)
            continue
        if char == "]":
            depth_bracket -= 1
            current.append(char)
            continue

        if char == "," and depth_paren == 0 and depth_brace == 0 and depth_bracket == 0:
            part = "".join(current).strip()
            if part:
                parts.append(part)
            current = []
            continue

        current.append(char)

    tail = "".join(current).strip()
    if tail:
        parts.append(tail)
    return parts


def unquote_string(text: str) -> str:
    text = text.strip()
    if len(text) >= 2 and text[0] == '"' and text[-1] == '"':
        return bytes(text[1:-1], "utf-8").decode("unicode_escape")
    if len(text) >= 2 and text[0] == "`" and text[-1] == "`":
        return text[1:-1]
    return text


def normalize_handler(text: str) -> str:
    text = re.sub(r"\s+", " ", text).strip()
    if text.startswith("func(") or text.startswith("func ("):
        return "inline route func"
    return text


def classify_section(file_path: str, full_path: str) -> str:
    if file_path in {"backend/internal/server/routes/common.go", "backend/internal/setup/handler.go"}:
        return "common"
    if file_path in AUTH_USER_FILES:
        return "auth-user"
    if file_path == "backend/internal/server/routes/gateway.go":
        return "gateway"
    if full_path.startswith("/api/v1/admin/ops"):
        return "admin-ops"
    if any(full_path.startswith(prefix) for prefix in ADMIN_CORE_PREFIXES):
        return "admin-core"
    if any(full_path.startswith(prefix) for prefix in ADMIN_ACCOUNTS_PREFIXES):
        return "admin-accounts"
    return "admin-misc"


def iter_candidate_statements(lines: list[str]) -> list[tuple[int, str]]:
    statements: list[tuple[int, str]] = []
    index = 0
    while index < len(lines):
        line = lines[index]
        stripped = line.lstrip()
        if stripped.startswith("//"):
            index += 1
            continue

        sanitized = remove_line_comment(line)
        if not GROUP_ASSIGN_RE.search(sanitized) and not ROUTE_CALL_RE.search(sanitized):
            index += 1
            continue

        start_line = index + 1
        parts = [line]
        balance = paren_delta(sanitized)

        while balance > 0 and index + 1 < len(lines):
            index += 1
            next_line = lines[index]
            parts.append(next_line)
            balance += paren_delta(remove_line_comment(next_line))

        statements.append((start_line, "".join(parts)))
        index += 1

    return statements


def parse_routes(repo_root: Path) -> list[Route]:
    routes: list[Route] = []

    for relative_path in ROUTE_FILES:
        file_path = repo_root / relative_path
        lines = file_path.read_text(encoding="utf-8").splitlines(keepends=True)
        group_paths: dict[str, str] = {"r": "", "v1": "/api/v1"}

        for line_number, statement in iter_candidate_statements(lines):
            group_match = GROUP_ASSIGN_RE.search(statement)
            if group_match:
                open_index = statement.find("(", group_match.end() - 1)
                if open_index != -1:
                    args = split_top_level_args(extract_call_args(statement, open_index))
                    if args:
                        group_paths[group_match.group("var")] = join_paths(
                            group_paths.get(group_match.group("base"), ""),
                            unquote_string(args[0]),
                        )
                continue

            route_match = ROUTE_CALL_RE.search(statement)
            if not route_match:
                continue

            open_index = statement.find("(", route_match.end() - 1)
            if open_index == -1:
                continue

            args = split_top_level_args(extract_call_args(statement, open_index))
            if not args:
                continue

            base_path = group_paths.get(route_match.group("var"), "")
            full_path = join_paths(base_path, unquote_string(args[0]))
            handler = normalize_handler(args[-1]) if len(args) > 1 else ""
            routes.append(
                Route(
                    method=route_match.group("method").upper(),
                    path=full_path,
                    handler=handler,
                    file=relative_path,
                    line=line_number,
                    section=classify_section(relative_path, full_path),
                )
            )

    return routes


def path_sort_key(path: str) -> str:
    # Match the repo's existing TSV ordering closely:
    # keep normal lexicographic path order, but place wildcard segments after
    # param segments by making "*" sort late.
    return path.replace("*", "~")


def sort_routes(routes: list[Route]) -> list[Route]:
    return sorted(
        routes,
        key=lambda route: (
            SECTION_ORDER[route.section],
            path_sort_key(route.path),
            route.method,
            ROUTE_FILES.index(route.file),
            route.line,
        ),
    )


def write_tsv(routes: list[Route], output_path: Path | None) -> None:
    rows = [
        ["method", "path", "handler", "file", "line", "section"],
        *[
            [route.method, route.path, route.handler, route.file, str(route.line), route.section]
            for route in routes
        ],
    ]

    if output_path is None:
        writer = csv.writer(sys.stdout, delimiter="\t", lineterminator="\n")
        writer.writerows(rows)
        return

    with output_path.open("w", encoding="utf-8", newline="") as fh:
        writer = csv.writer(fh, delimiter="\t", lineterminator="\n")
        writer.writerows(rows)


def write_summary_json(routes: list[Route], output_path: Path) -> None:
    counts = {
        "common": 0,
        "auth-user": 0,
        "gateway": 0,
        "admin-core": 0,
        "admin-accounts": 0,
        "admin-ops": 0,
        "admin-misc": 0,
    }
    for route in routes:
        counts[route.section] += 1

    payload = {
        "synced_at": str(date.today()),
        "version": "api_full_reference_v2",
        "policy": "summary_manifest_only",
        "totals": {
            "all_routes": len(routes),
            "common_setup": counts["common"],
            "auth_user": counts["auth-user"],
            "gateway": counts["gateway"],
            "admin_core": counts["admin-core"],
            "admin_accounts": counts["admin-accounts"],
            "admin_ops": counts["admin-ops"],
            "admin_misc": counts["admin-misc"],
            "admin_key_supported": sum(1 for route in routes if route.path.startswith("/api/v1/admin/")),
        },
        "source_files": ROUTE_FILES,
        "notes": [
            "This file is a lightweight machine-readable manifest for the synced documentation set.",
            "Detailed route tables live in 01-common-setup.md through 07-admin-misc.md.",
            "All /api/v1/admin/** routes are summarized in 08-admin-key-supported.md.",
        ],
    }

    output_path.write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


def write_unresolved(routes: list[Route], output_path: Path) -> None:
    inline_routes = [route for route in routes if route.handler == "inline route func"]

    lines = [
        "当前版本未保留旧版的“逐 handler 字段级展开”提取结果。",
        "",
        "本次同步优先保证：",
        "- 路由覆盖完整",
        "- 方法 / 路径准确",
        "- 鉴权与 admin-key 结论准确",
        "- 注册位置准确",
        "",
        "下列路由使用 inline wrapper，而不是直接注册某个 handler 方法：",
    ]

    for route in inline_routes:
        lines.append(f"- {route.method} {route.path}")

    lines.extend(
        [
            "",
            "这些路由已在对应 Markdown 表格的“Handler”与“备注”列中说明，不再单独视为 unresolved。",
        ]
    )

    output_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Export Sub2API routes from Go route-registration files."
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="Repository root. Defaults to the current working directory.",
    )
    parser.add_argument(
        "--tsv-output",
        help="Write TSV output to this path. Defaults to stdout.",
    )
    parser.add_argument(
        "--summary-json",
        help="Optional path for endpoints.json-style summary output.",
    )
    parser.add_argument(
        "--unresolved-output",
        help="Optional path for UNRESOLVED_HANDLERS.txt-style output.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    repo_root = Path(args.repo_root).resolve()
    routes = sort_routes(parse_routes(repo_root))

    tsv_output = Path(args.tsv_output) if args.tsv_output else None
    write_tsv(routes, tsv_output)

    if args.summary_json:
        write_summary_json(routes, Path(args.summary_json))
    if args.unresolved_output:
        write_unresolved(routes, Path(args.unresolved_output))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
