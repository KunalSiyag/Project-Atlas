#!/usr/bin/env python3
"""Verify Project Atlas M0 documentation foundation is present and substantive.

This is the durable structural test for the documentation-first foundation.
It fails if required durable docs are missing, lack standard metadata sections,
or are headers-only stubs.
"""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
META = [
    "Purpose",
    "Scope",
    "Dependencies",
    "Owner",
    "Status",
    "Last Updated",
    "Related Documents",
]

PRODUCT = [
    "VISION.md",
    "GOALS.md",
    "NON_GOALS.md",
    "PERSONAS.md",
    "FUNCTIONAL_REQUIREMENTS.md",
    "NON_FUNCTIONAL_REQUIREMENTS.md",
    "SUCCESS_METRICS.md",
    "GLOSSARY.md",
]

DOMAIN = [
    "workspace.md",
    "agent.md",
    "conversation.md",
    "memory.md",
    "tool.md",
    "provider.md",
    "evaluation.md",
    "prompt.md",
    "workflow.md",
]

ARCHITECTURE = [
    "OVERVIEW.md",
    "gateway.md",
    "planner.md",
    "memory.md",
    "context-builder.md",
    "inference.md",
    "tool-runtime.md",
    "storage.md",
    "security.md",
    "observability.md",
    "evaluation.md",
    "deployment.md",
]

ARCH_BODY = [
    "Why It Exists",
    "Problem Solved",
    "Consumers",
    "Public Interfaces",
    "Dependencies",
    "Failure Modes",
]

ADR_GLOBS = [
    "0001-*.md",
    "0002-*.md",
    "0003-*.md",
    "0004-*.md",
    "0005-*.md",
    "0006-*.md",
    "0007-*.md",
]

ADR_BODY = ["Context", "Options", "Decision", "Consequences"]

NAV = [
    "management/ROADMAP.md",
    "management/MILESTONES.md",
    "management/SPRINTS.md",
    "management/BACKLOG.md",
    "management/CHANGELOG.md",
    "contracts/README.md",
    "schemas/README.md",
    "tasks/README.md",
    "README.md",
    "AI_CONTEXT.md",
]


def fail(msg: str, errors: list[str]) -> None:
    errors.append(msg)


def check_meta(path: Path, text: str, errors: list[str]) -> None:
    for field in META:
        if f"## {field}" not in text:
            fail(f"{path.relative_to(ROOT)}: missing metadata section ## {field}", errors)


def check_substantive(path: Path, text: str, errors: list[str], min_bytes: int = 400) -> None:
    if len(text) < min_bytes:
        fail(f"{path.relative_to(ROOT)}: too short ({len(text)} bytes); likely stub", errors)
    if text.count("\n") < 15:
        fail(f"{path.relative_to(ROOT)}: too few lines; likely stub", errors)


def main() -> int:
    errors: list[str] = []

    for name in PRODUCT:
        path = ROOT / "docs" / "product" / name
        if not path.is_file():
            fail(f"missing product doc {path.relative_to(ROOT)}", errors)
            continue
        text = path.read_text(encoding="utf-8")
        check_meta(path, text, errors)
        check_substantive(path, text, errors)

    for name in DOMAIN:
        path = ROOT / "docs" / "domain" / name
        if not path.is_file():
            fail(f"missing domain doc {path.relative_to(ROOT)}", errors)
            continue
        text = path.read_text(encoding="utf-8")
        check_meta(path, text, errors)
        check_substantive(path, text, errors)
        if "## Definition" not in text:
            fail(f"{path.relative_to(ROOT)}: missing ## Definition", errors)
        if "## Relationships" not in text:
            fail(f"{path.relative_to(ROOT)}: missing ## Relationships", errors)

    for name in ARCHITECTURE:
        path = ROOT / "docs" / "architecture" / name
        if not path.is_file():
            fail(f"missing architecture doc {path.relative_to(ROOT)}", errors)
            continue
        text = path.read_text(encoding="utf-8")
        check_meta(path, text, errors)
        check_substantive(path, text, errors)
        if name != "OVERVIEW.md":
            for section in ARCH_BODY:
                if f"## {section}" not in text:
                    fail(f"{path.relative_to(ROOT)}: missing ## {section}", errors)
        elif "Subsystem Index" not in text:
            fail(f"{path.relative_to(ROOT)}: missing Subsystem Index", errors)

    adr_dir = ROOT / "docs" / "adr"
    for pattern in ADR_GLOBS:
        matches = list(adr_dir.glob(pattern))
        if not matches:
            fail(f"missing ADR matching {pattern}", errors)
            continue
        path = matches[0]
        text = path.read_text(encoding="utf-8")
        check_meta(path, text, errors)
        check_substantive(path, text, errors)
        for section in ADR_BODY:
            if f"## {section}" not in text:
                fail(f"{path.relative_to(ROOT)}: missing ## {section}", errors)

    for rel in NAV:
        path = ROOT / rel
        if not path.is_file():
            fail(f"missing nav doc {rel}", errors)
            continue
        text = path.read_text(encoding="utf-8")
        if len(text) < 200:
            fail(f"{rel}: too short for navigation doc", errors)

    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    ai = (ROOT / "AI_CONTEXT.md").read_text(encoding="utf-8")
    if "M0" not in readme or "complete" not in readme.lower():
        fail("README.md: missing M0 complete orientation", errors)
    if "M0" not in ai:
        fail("AI_CONTEXT.md: missing M0 status", errors)
    if "M1" not in readme and "M1" not in ai:
        fail("README/AI_CONTEXT: missing next-step (M1) orientation", errors)

    # M0 must not require application Go code
    go_files = [p for p in ROOT.rglob("*.go") if ".git" not in p.parts]
    if go_files:
        # Not a failure: implementation may exist later. Record only.
        print(f"note: found {len(go_files)} .go files (not required for M0)")

    if errors:
        print("M0 foundation verification FAILED:")
        for e in errors:
            print(f"  - {e}")
        return 1

    print("M0 foundation verification PASSED")
    print(f"  product docs: {len(PRODUCT)}")
    print(f"  domain docs: {len(DOMAIN)}")
    print(f"  architecture docs: {len(ARCHITECTURE)}")
    print(f"  foundation ADRs checked: {len(ADR_GLOBS)}")
    print(f"  nav docs: {len(NAV)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
