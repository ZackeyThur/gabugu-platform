#!/usr/bin/env python3
"""Verify imported production source bytes and an optional published Git ref.

This is a read-only verifier. Runtime identities in the baseline are records,
not live observations. It never deploys, rebuilds, or restores data.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import subprocess
import sys
from typing import Any

try:
    from .release_live import LiveVerificationError, observe_vps
except ImportError:
    from release_live import LiveVerificationError, observe_vps


class VerificationError(Exception):
    """An incomplete or inconsistent synchronization baseline."""


SHA256 = re.compile(r"[0-9a-fA-F]{64}\Z")
IMAGE_DIGEST = re.compile(r"sha256:[0-9a-fA-F]{64}\Z")
SUM_LINE = re.compile(r"([0-9a-fA-F]{64}) [ *](.+)\Z")
WINDOWS_RESERVED = re.compile(r"(?:CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(?:\..*)?\Z", re.I)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise VerificationError(message)


def relative_path(value: Any, label: str) -> PurePosixPath:
    """Use one unambiguous, portable relative path format on both platforms."""
    require(isinstance(value, str) and bool(value), f"{label}: missing relative path")
    require(not any(ord(char) < 32 for char in value), f"{label}: control character in path")
    require(not any(char in value for char in '\\:*?"<>|'), f"{label}: non-portable path")
    parts = value.split("/")
    require(all(part and part not in (".", "..") for part in parts), f"{label}: unsafe path component")
    require(all(not part.endswith((" ", ".")) and not WINDOWS_RESERVED.fullmatch(part)
                for part in parts), f"{label}: non-portable path component")
    result = PurePosixPath(value)
    require(not result.is_absolute(), f"{label}: absolute path")
    return result


def is_link(info: os.stat_result) -> bool:
    return stat.S_ISLNK(info.st_mode) or bool(
        getattr(info, "st_file_attributes", 0) & getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 1024)
    )


def checked_path(root: Path, value: Any, label: str) -> Path:
    rel = relative_path(value, label)
    current = root
    for part in rel.parts:
        current = current / part
        try:
            info = current.lstat()
        except OSError:
            raise VerificationError(f"{label}: path unavailable") from None
        require(not is_link(info), f"{label}: symlink or reparse point")
    require(current.resolve().is_relative_to(root), f"{label}: path escapes repository")
    return current


def sha256_file(path: Path) -> str:
    hasher = hashlib.sha256()
    try:
        with path.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                hasher.update(chunk)
    except OSError:
        raise VerificationError("file could not be read") from None
    return hasher.hexdigest()


def digest(value: Any, label: str) -> str:
    require(isinstance(value, str) and SHA256.fullmatch(value) is not None,
            f"{label}: expected SHA256")
    return value.lower()


def unique_case_path(seen: dict[str, str], path: str, label: str) -> None:
    key = path.casefold()
    require(key not in seen, f"{label}: duplicate or case-conflicting path")
    seen[key] = path


def read_manifest(path: Path, expected_sha: Any, label: str) -> dict[str, str]:
    require(path.is_file(), f"{label}: manifest is not a file")
    require(sha256_file(path) == digest(expected_sha, label), f"{label}: manifest digest mismatch")
    try:
        content = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError):
        raise VerificationError(f"{label}: manifest must be UTF-8") from None
    entries: dict[str, str] = {}
    seen: dict[str, str] = {}
    for line_number, line in enumerate(content.splitlines(), 1):
        match = SUM_LINE.fullmatch(line)
        require(match is not None, f"{label}: invalid checksum line {line_number}")
        checksum, name = match.groups()
        relative_path(name, f"{label} line {line_number}")
        unique_case_path(seen, name, label)
        entries[name] = checksum.lower()
    require(bool(entries), f"{label}: empty manifest")
    return entries


def actual_files(root: Path, label: str) -> dict[str, Path]:
    result: dict[str, Path] = {}
    seen: dict[str, str] = {}

    def visit(directory: Path) -> None:
        try:
            children = sorted(directory.iterdir(), key=lambda item: item.name)
        except OSError:
            raise VerificationError(f"{label}: directory could not be read") from None
        for child in children:
            rel = child.relative_to(root).as_posix()
            relative_path(rel, label)
            unique_case_path(seen, rel, label)
            try:
                info = child.lstat()
            except OSError:
                raise VerificationError(f"{label}: file disappeared during verification") from None
            require(not is_link(info), f"{label}: symlink or reparse point")
            if stat.S_ISDIR(info.st_mode):
                visit(child)
            else:
                require(stat.S_ISREG(info.st_mode), f"{label}: non-regular file")
                result[rel] = child

    visit(root)
    return result


def verify_runtime(runtime: Any) -> None:
    require(isinstance(runtime, dict), "runtime: expected object")
    backends = runtime.get("backend")
    require(isinstance(backends, list) and bool(backends), "runtime.backend: expected nonempty list")
    for index, backend in enumerate(backends):
        label = f"runtime.backend[{index}]"
        require(isinstance(backend, dict), f"{label}: expected object")
        for field in ("source_manifest_sha256", "archive_sha256", "binary_sha256"):
            digest(backend.get(field), f"{label}.{field}")
        require(isinstance(backend.get("image_digest"), str) and
                IMAGE_DIGEST.fullmatch(backend["image_digest"]) is not None,
                f"{label}.image_digest: expected immutable image digest")
        if "container_binary_sha256" in backend:
            require(digest(backend["container_binary_sha256"], label) ==
                    digest(backend["binary_sha256"], label), f"{label}: host/container binary mismatch")
    frontend = runtime.get("frontend")
    require(isinstance(frontend, dict), "runtime.frontend: expected object")
    for field in ("source_manifest_sha256", "dist_manifest_sha256"):
        digest(frontend.get(field), f"runtime.frontend.{field}")
    if "archive_sha256" in frontend:
        digest(frontend["archive_sha256"], "runtime.frontend.archive_sha256")


def verify_local(repo_root: Path, baseline: Any, ops_root: Path | None = None) -> list[dict[str, Any]]:
    """Check all mapped files, including extra files, using their original bytes."""
    root = repo_root.resolve()
    require(root.is_dir(), "repository root unavailable")
    manifest_root = (ops_root if ops_root is not None else repo_root).resolve()
    require(manifest_root.is_dir(), "ops root unavailable")
    require(isinstance(baseline, dict) and baseline.get("schema_version") == 1,
            "unsupported baseline schema_version")
    require(isinstance(baseline.get("release_id"), str) and bool(baseline["release_id"]),
            "release_id: missing")
    verify_runtime(baseline.get("runtime"))
    mappings = baseline.get("source_mappings")
    require(isinstance(mappings, list) and bool(mappings), "source_mappings: expected nonempty list")
    roots: list[PurePosixPath] = []
    ids: set[str] = set()
    reports: list[dict[str, Any]] = []
    for index, mapping in enumerate(mappings):
        label = f"source_mappings[{index}]"
        require(isinstance(mapping, dict), f"{label}: expected object")
        mapping_id = mapping.get("id")
        require(isinstance(mapping_id, str) and bool(mapping_id) and mapping_id not in ids,
                f"{label}: missing or duplicate id")
        ids.add(mapping_id)
        rel_root = relative_path(mapping.get("root"), label)
        folded = PurePosixPath(rel_root.as_posix().casefold())
        require(all(folded != other and not folded.is_relative_to(other) and
                    not other.is_relative_to(folded) for other in roots), f"{label}: overlapping mapping roots")
        roots.append(folded)
        source = checked_path(root, mapping["root"], label)
        require(source.is_dir(), f"{label}: root is not a directory")
        manifest = checked_path(manifest_root, mapping.get("manifest"), label)
        entries = read_manifest(manifest, mapping.get("manifest_sha256"), label)
        actual = actual_files(source, label)
        missing = entries.keys() - actual.keys()
        extra = actual.keys() - entries.keys()
        require(not missing and not extra,
                f"{label}: file set mismatch (missing={len(missing)}, extra={len(extra)})")
        for name, checksum in entries.items():
            require(sha256_file(actual[name]) == checksum, f"{label}: source digest mismatch")
        reports.append({"mapping": mapping_id, "files": len(entries), "verified": True})
    artifacts = baseline.get("artifact_checks", [])
    require(isinstance(artifacts, list), "artifact_checks: expected list")
    for index, artifact in enumerate(artifacts):
        label = f"artifact_checks[{index}]"
        require(isinstance(artifact, dict), f"{label}: expected object")
        path = checked_path(root, artifact.get("path"), label)
        require(path.is_file(), f"{label}: not a file")
        require(sha256_file(path) == digest(artifact.get("sha256"), label), f"{label}: artifact digest mismatch")
    return reports


def git_output(root: Path, arguments: list[str]) -> str:
    try:
        result = subprocess.run(["git", "-C", str(root), *arguments], capture_output=True,
                                text=True, timeout=60, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise VerificationError("Git command unavailable or timed out") from None
    require(result.returncode == 0, "Git command failed (details suppressed to protect credentials)")
    return result.stdout.strip()


def verify_remote_ref(root: Path, remote: str, ref: str) -> str:
    require(isinstance(remote, str) and re.fullmatch(r"[A-Za-z0-9_.-]+", remote) is not None and
            not remote.startswith("-"), "remote must be a configured Git remote name")
    require(isinstance(ref, str) and ref.startswith(("refs/heads/", "refs/tags/")),
            "remote ref must be a full branch or tag ref")
    git_output(root, ["check-ref-format", ref])
    require(not git_output(root, ["status", "--porcelain", "--untracked-files=all"]),
            "remote synchronization requires a clean local worktree")
    head = git_output(root, ["rev-parse", "HEAD"])
    output = git_output(root, ["ls-remote", "--exit-code", remote, ref, ref + "^{}"])
    matches = {}
    for line in output.splitlines():
        parts = line.split("\t")
        require(len(parts) == 2 and re.fullmatch(r"[0-9a-f]{40,64}", parts[0]) is not None,
                "invalid remote ref response")
        if parts[1] in (ref, ref + "^{}"):
            require(parts[1] not in matches, "ambiguous remote ref response")
            matches[parts[1]] = parts[0]
    require(ref in matches, "production remote ref missing")
    require(matches.get(ref + "^{}", matches[ref]) == head, "production remote ref differs from local HEAD")
    return head


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["verify"])
    parser.add_argument("--repo-root", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--ops-root", type=Path, default=None,
                        help="Optional external ops root holding baseline JSON and manifest files")
    parser.add_argument("--baseline", default="platform/deploy/production-source.json")
    parser.add_argument("--remote-ref", help="Full production branch/tag ref; checks clean tree and HEAD")
    parser.add_argument("--remote", help="Configured Git remote name (default baseline git.remote or origin)")
    parser.add_argument("--live-vps", help="SSH alias for read-only live VPS observer")
    args = parser.parse_args(argv)
    try:
        root = args.repo_root.resolve()
        ops_root = (args.ops_root if args.ops_root is not None else root).resolve()
        require(ops_root.is_dir(), "ops root unavailable")
        baseline_path = checked_path(ops_root, args.baseline, "baseline")
        require(baseline_path.is_file(), "baseline: not a file")
        try:
            baseline = json.loads(baseline_path.read_text(encoding="utf-8"))
        except (OSError, UnicodeError, json.JSONDecodeError):
            raise VerificationError("baseline: unreadable or invalid JSON") from None
        reports = verify_local(root, baseline, ops_root=ops_root)
        head = None
        if args.remote_ref:
            git_config = baseline.get("git", {})
            require(isinstance(git_config, dict), "git: expected object")
            head = verify_remote_ref(root, args.remote or git_config.get("remote", "origin"), args.remote_ref)
        live = None
        if args.live_vps:
            live = observe_vps(args.live_vps, baseline)
        print(json.dumps({"verified": True, "source_mappings": reports, "remote_head": head,
                          "runtime_live_verified": bool(live), "live": live}, ensure_ascii=True))
        return 0
    except (VerificationError, LiveVerificationError) as error:
        print(f"verification failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
