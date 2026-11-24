"""Behavioral tests using temporary files and local Git repositories only."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import release_sync as sync
import release_live as live
import base64
import contextlib
import io
import os


class SourceVerificationTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "source").mkdir()
        (self.root / "source/file.txt").write_bytes(b"original\r\nbytes\x00")
        data_sha = hashlib.sha256((self.root / "source/file.txt").read_bytes()).hexdigest()
        (self.root / "manifest.txt").write_bytes(f"{data_sha}  file.txt\n".encode())
        checksum = "a" * 64
        self.baseline = {
            "schema_version": 1, "release_id": "test",
            "source_mappings": [{"id": "backend", "root": "source", "manifest": "manifest.txt",
                                 "manifest_sha256": sync.sha256_file(self.root / "manifest.txt")}],
            "runtime": {"backend": [{"source_manifest_sha256": checksum, "archive_sha256": checksum,
                                     "binary_sha256": checksum, "image_digest": f"sha256:{checksum}"}],
                        "frontend": {"source_manifest_sha256": checksum, "dist_manifest_sha256": checksum}},
        }

    def verify(self):
        return sync.verify_local(self.root, self.baseline)

    def set_manifest(self, text):
        (self.root / "manifest.txt").write_bytes(text.encode())
        self.baseline["source_mappings"][0]["manifest_sha256"] = sync.sha256_file(self.root / "manifest.txt")

    def test_raw_bytes_pass_then_line_endings_fail(self):
        self.assertEqual(self.verify()[0]["files"], 1)
        (self.root / "source/file.txt").write_bytes(b"original\nbytes\x00")
        with self.assertRaisesRegex(sync.VerificationError, "source digest mismatch"):
            self.verify()

    def test_extra_and_missing_files_fail(self):
        (self.root / "source/extra.txt").write_bytes(b"unexpected")
        with self.assertRaisesRegex(sync.VerificationError, "extra=1"):
            self.verify()
        (self.root / "source/extra.txt").unlink()
        (self.root / "source/file.txt").unlink()
        with self.assertRaisesRegex(sync.VerificationError, "missing=1"):
            self.verify()

    def test_manifest_tampering_fails_before_file_checks(self):
        (self.root / "manifest.txt").write_bytes(b"changed")
        with self.assertRaisesRegex(sync.VerificationError, "manifest digest mismatch"):
            self.verify()

    def test_unsafe_paths_rejected(self):
        for value in ("../secret", "/absolute", "C:/secret", "a\\b", "./file", "a//b", "nul.txt", "file. "):
            with self.subTest(path=value), self.assertRaises(sync.VerificationError):
                sync.relative_path(value, "test")
        self.set_manifest("a" * 64 + "  ../secret\n")
        with self.assertRaisesRegex(sync.VerificationError, "unsafe path"):
            self.verify()

    def test_duplicate_and_case_conflicting_manifest_fails(self):
        for other in ("file.txt", "FILE.txt"):
            self.set_manifest("a" * 64 + "  file.txt\n" + "b" * 64 + f"  {other}\n")
            with self.subTest(other=other), self.assertRaisesRegex(sync.VerificationError, "case-conflicting"):
                self.verify()

    def test_symlink_rejected_even_when_extra(self):
        try:
            (self.root / "source/link").symlink_to(self.root / "manifest.txt")
        except OSError as error:
            self.skipTest(f"symlink unavailable: {error.__class__.__name__}")
        with self.assertRaisesRegex(sync.VerificationError, "symlink"):
            self.verify()

    def test_mapping_root_symlink_rejected(self):
        try:
            (self.root / "alias").symlink_to(self.root / "source", target_is_directory=True)
        except OSError as error:
            self.skipTest(f"symlink unavailable: {error.__class__.__name__}")
        self.baseline["source_mappings"][0]["root"] = "alias"
        with self.assertRaisesRegex(sync.VerificationError, "symlink"):
            self.verify()

    def test_host_container_binary_mismatch_fails(self):
        self.baseline["runtime"]["backend"][0]["container_binary_sha256"] = "b" * 64
        with self.assertRaisesRegex(sync.VerificationError, "host/container binary mismatch"):
            self.verify()

    def test_local_binary_artifact_is_hashed(self):
        (self.root / "binary").write_bytes(b"fixed binary")
        self.baseline["artifact_checks"] = [{"id": "binary", "path": "binary",
                                            "sha256": sync.sha256_file(self.root / "binary")}]
        self.verify()
        (self.root / "binary").write_bytes(b"other binary")
        with self.assertRaisesRegex(sync.VerificationError, "artifact digest mismatch"):
            self.verify()

    def test_backend_records_can_have_distinct_identities(self):
        self.baseline["runtime"]["backend"].append({
            "source_manifest_sha256": "b" * 64, "archive_sha256": "c" * 64,
            "binary_sha256": "d" * 64, "image_digest": "sha256:" + "e" * 64,
        })
        self.verify()

    def test_cli_returns_nonzero_for_incomplete_sync(self):
        (self.root / "baseline.json").write_text(json.dumps(self.baseline), encoding="utf-8")
        (self.root / "source/file.txt").write_bytes(b"changed")
        with mock.patch("sys.stderr"):
            self.assertEqual(sync.main(["verify", "--repo-root", str(self.root), "--baseline", "baseline.json"]), 1)

    def test_external_ops_root_verifies_split_repo_and_rejects_escape(self):
        with tempfile.TemporaryDirectory() as ops_tmp:
            ops_root = Path(ops_tmp)
            (ops_root / "manifests").mkdir()
            data_sha = hashlib.sha256((self.root / "source/file.txt").read_bytes()).hexdigest()
            manifest_path = ops_root / "manifests/backend.sha256"
            manifest_path.write_bytes(f"{data_sha}  file.txt\n".encode())
            (self.root / "LICENSE").write_bytes(b"MIT")
            split_baseline = {
                "schema_version": 1, "release_id": "split-test",
                "source_mappings": [{"id": "backend", "root": "source",
                                     "manifest": "manifests/backend.sha256",
                                     "manifest_sha256": sync.sha256_file(manifest_path)}],
                "runtime": self.baseline["runtime"],
                "artifact_checks": [{"id": "license", "path": "LICENSE",
                                     "sha256": sync.sha256_file(self.root / "LICENSE")}],
            }
            (ops_root / "production-source.json").write_text(json.dumps(split_baseline), encoding="utf-8")
            with contextlib.redirect_stdout(io.StringIO()) as out:
                rc = sync.main([
                    "verify", "--repo-root", str(self.root),
                    "--ops-root", str(ops_root),
                    "--baseline", "production-source.json",
                ])
            self.assertEqual(rc, 0)
            self.assertTrue(json.loads(out.getvalue())["verified"])
            split_baseline["source_mappings"][0]["manifest"] = "../outside.sha256"
            with self.assertRaisesRegex(sync.VerificationError, "unsafe path"):
                sync.verify_local(self.root, split_baseline, ops_root=ops_root)


class RemoteVerificationTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "local"
        self.root.mkdir()
        self.remote = Path(self.temp.name) / "remote.git"
        self.git("init", "-q")
        self.git("config", "user.name", "Verifier Test")
        self.git("config", "user.email", "verifier@example.invalid")
        self.git("-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "initial")
        subprocess.run(["git", "init", "--bare", "-q", str(self.remote)], check=True, capture_output=True)
        self.git("remote", "add", "origin", str(self.remote))
        self.git("push", "-q", "origin", "HEAD:refs/heads/production")

    def git(self, *args):
        result = subprocess.run(["git", "-C", str(self.root), *args], check=True, capture_output=True, text=True)
        return result.stdout.strip()

    def test_matching_ref_passes_then_new_local_commit_fails(self):
        self.assertEqual(sync.verify_remote_ref(self.root, "origin", "refs/heads/production"),
                         self.git("rev-parse", "HEAD"))
        self.git("-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "next")
        with self.assertRaisesRegex(sync.VerificationError, "differs from local HEAD"):
            sync.verify_remote_ref(self.root, "origin", "refs/heads/production")

    def test_dirty_worktree_fails_even_when_remote_commit_matches(self):
        (self.root / "extra.txt").write_bytes(b"not committed")
        with self.assertRaisesRegex(sync.VerificationError, "clean local worktree"):
            sync.verify_remote_ref(self.root, "origin", "refs/heads/production")

    def test_annotated_tag_uses_peeled_commit(self):
        self.git("-c", "tag.gpgsign=false", "tag", "-a", "production-test", "-m", "release")
        self.git("push", "-q", "origin", "refs/tags/production-test")
        self.assertEqual(sync.verify_remote_ref(self.root, "origin", "refs/tags/production-test"),
                         self.git("rev-parse", "HEAD"))

    def test_missing_remote_ref_fails(self):
        with self.assertRaises(sync.VerificationError):
            sync.verify_remote_ref(self.root, "origin", "refs/heads/missing")


class LiveVerificationTest(unittest.TestCase):
    def expected(self):
        return {"runtime": {"backend": [{"release_path": "/opt/gabugu/releases/backend-test",
                "container_name": "backend-test", "upstream_address": "127.0.0.1:18123", "version": "test-version"}],
                "frontend": {"release_path": "/opt/gabugu/releases/frontend-test",
                "nginx_root": "/var/www/gabugu-frontend/frontend-test",
                "hostname": "gabugu.com", "manifest_filename": "manifest.json", "index_asset": "/assets/current.js"}}}

    def observer_namespace(self, baseline):
        payload = base64.b64encode(json.dumps(baseline).encode()).decode()
        source = live.observer_source().replace("__PAYLOAD__", payload).split("\ntry:\n    main()")[0]
        namespace = {}
        exec(compile(source, "observer-test", "exec"), namespace)
        return namespace

    def test_ssh_alias_injection_and_unsafe_roots_rejected_before_subprocess(self):
        with mock.patch("release_live.subprocess.run") as run:
            for alias in ("-oProxyCommand=bad", "vps;echo bad", "vps bad", "-vps"):
                with self.subTest(alias=alias), self.assertRaises(live.LiveVerificationError):
                    live.observe_vps(alias, self.expected())
            bad = self.expected()
            bad["runtime"]["frontend"]["nginx_root"] = "/var/www/gabugu-frontend/../secret"
            with self.assertRaises(live.LiveVerificationError):
                live.observe_vps("vps", bad)
            run.assert_not_called()

    def test_ssh_uses_stdin_script_and_requires_live_true(self):
        with mock.patch("release_live.subprocess.run", return_value=subprocess.CompletedProcess(
                [], 0, '{"runtime_live_verified":true}', "")) as run:
            self.assertTrue(live.observe_vps("vps", self.expected())["runtime_live_verified"])
            self.assertEqual(run.call_args.args[0], ["ssh", "vps", "python3", "-"])
            self.assertIn("base64.b64decode", run.call_args.kwargs["input"])
        with mock.patch("release_live.subprocess.run", return_value=subprocess.CompletedProcess(
                [], 1, '{"runtime_live_verified":false,"mismatch":"changed"}', "secret")):
            with self.assertRaisesRegex(live.LiveVerificationError, "differs"):
                live.observe_vps("vps", self.expected())

    def test_nginx_named_upstream_and_matching_hostname(self):
        namespace = self.observer_namespace({})
        config = """http {
            upstream gabugu_active { server 127.0.0.1:18123; }
            server { listen 80; server_name gabugu.com; root /var/www/html; }
            server { listen 443 ssl; server_name other.example; root /wrong; }
            server { listen 443 ssl; server_name gabugu.com; root /expected;
                location /api { proxy_pass http://gabugu_active; }
            }
        }"""
        self.assertEqual(namespace["parse_nginx"](config, "gabugu.com"),
                         (["/expected"], ["127.0.0.1:18123"]))

    def test_manifest_on_fake_root_checks_raw_bytes_and_extra_file(self):
        namespace = self.observer_namespace({})
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            (root / "file").write_bytes(b"raw\r\n")
            (root / "frontend-dist-manifest.json").write_bytes(b"[]")
            manifest = root / "source-manifest.json"
            manifest.write_text(json.dumps([{"path": "file", "sha256": sync.sha256_file(root / "file")}]))
            checksum = sync.sha256_file(manifest)
            check = namespace["check_manifest"]
            self.assertEqual(check(str(root), str(manifest), checksum, "backend"), 1)
            (root / "file").write_bytes(b"raw\n")
            with self.assertRaisesRegex(namespace["E"], "source digest mismatch"):
                check(str(root), str(manifest), checksum, "backend")
            (root / "file").write_bytes(b"raw\r\n")
            (root / "extra").write_bytes(b"extra")
            with self.assertRaisesRegex(namespace["E"], "file set differs"):
                check(str(root), str(manifest), checksum, "backend")

    def test_observer_actual_binary_and_changed_nginx_root_fail(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            backend = root / "backend"; source = backend / "source"; source.mkdir(parents=True)
            frontend = root / "frontend"; frontend.mkdir()
            dist = root / "dist"; dist.mkdir()
            (source / "file").write_bytes(b"backend")
            (source / "frontend-dist-manifest.json").write_bytes(b"[]")
            (source / "source-manifest.json").write_text(json.dumps([
                {"path": "file", "sha256": sync.sha256_file(source / "file")}]))
            (dist / "index.html").write_bytes(b"fixed nonce")
            (dist / "assets").mkdir()
            (dist / "assets/current.js").write_bytes(b"current script")
            fm = {"source_archive_sha256": "a" * 64,
                  "assets": [{"path": "index.html", "sha256": sync.sha256_file(dist / "index.html")},
                             {"path": "assets/current.js", "sha256": sync.sha256_file(dist / "assets/current.js")} ]}
            (frontend / "manifest.json").write_text(json.dumps(fm))
            base = {"runtime": {"backend": [{"release_path": str(backend), "container_name": "test",
                "source_manifest_sha256": sync.sha256_file(source / "source-manifest.json"),
                "image_digest": "sha256:" + "a" * 64, "binary_sha256": "b" * 64,
                "upstream_address": "127.0.0.1:18123", "version": "test-version"}], "frontend": {"release_path": str(frontend),
                "nginx_root": str(dist), "manifest_filename": "manifest.json", "archive_sha256": "a" * 64,
                "dist_manifest_sha256": sync.sha256_file(frontend / "manifest.json"),
                "source_manifest_sha256": sync.sha256_file(frontend / "manifest.json"), "hostname": "gabugu.com",
                "index_asset": "/assets/current.js"}}}
            namespace = self.observer_namespace(base)
            namespace["safe_root"] = lambda p: p  # Linux-only /-root traversal; fixture roots already trusted.
            original_hash = namespace["hfile"]
            namespace["hfile"] = lambda p: "b" * 64 if p == "/proc/123/exe" else original_hash(p)
            docker = {"Image": "sha256:" + "a" * 64, "RestartCount": 0,
                "State": {"Pid": 123, "Health": {"Status": "healthy"}},
                "NetworkSettings": {"Ports": {"8080/tcp": [{"HostIp": "127.0.0.1", "HostPort": "18123"}]}}}
            nginx = "http { upstream active { server 127.0.0.1:18123; } server { listen 443 ssl; server_name gabugu.com; root " + str(dist) + "; location /api { proxy_pass http://active; } } }"
            namespace["cmd"] = lambda *args: json.dumps(docker) if args[0] == "docker" else nginx
            namespace["https_fetch"] = lambda host, path: (
                b'<script src="/assets/current.js"></script>' if path == "/" else
                b'{"data":{"version":"test-version"}}' if path == "/api/v1/settings/public" else b"current script")
            original_realpath = os.path.realpath
            with mock.patch("os.path.realpath", side_effect=lambda p: str(backend) if p == "/opt/gabugu/current-release" else original_realpath(p)):
                with contextlib.redirect_stdout(io.StringIO()) as output:
                    namespace["main"]()
                self.assertTrue(json.loads(output.getvalue())["runtime_live_verified"])
                namespace["hfile"] = lambda p: "c" * 64 if p == "/proc/123/exe" else original_hash(p)
                with self.assertRaisesRegex(namespace["E"], "running binary digest differs"):
                    namespace["main"]()
                namespace["hfile"] = lambda p: "b" * 64 if p == "/proc/123/exe" else original_hash(p)
                nginx = nginx.replace(str(dist), "/wrong")
                with self.assertRaisesRegex(namespace["E"], "nginx active root differs"):
                    namespace["main"]()

    def test_loaded_https_rejects_stale_asset_index_and_api_version(self):
        namespace = self.observer_namespace({})
        front = {"hostname": "gabugu.com", "index_asset": "/assets/current.js"}
        manifest = {"assets": [{"path": "assets/current.js", "sha256": hashlib.sha256(b"current").hexdigest()},
                               {"path": "index.html", "sha256": "a" * 64}]}
        responses = {"/assets/current.js": b"current", "/": b'<script src="/assets/current.js" nonce="dynamic"></script>',
                     "/api/v1/settings/public": b'{"data":{"version":"test-version"}}'}
        namespace["https_fetch"] = lambda host, path: responses[path]
        self.assertEqual(namespace["check_loaded_https"](front, {"version": "test-version"}, manifest), 1)
        responses["/assets/current.js"] = b"stale"
        with self.assertRaisesRegex(namespace["E"], "HTTPS asset digest differs"):
            namespace["check_loaded_https"](front, {"version": "test-version"}, manifest)
        responses["/assets/current.js"] = b"current"
        responses["/"] = b'<script src="/assets/old.js"></script>'
        with self.assertRaisesRegex(namespace["E"], "index entrypoint differs"):
            namespace["check_loaded_https"](front, {"version": "test-version"}, manifest)
        responses["/"] = b'<script src="/assets/current.js"></script>'
        responses["/api/v1/settings/public"] = b'{"data":{"version":"old"}}'
        with self.assertRaisesRegex(namespace["E"], "backend version differs"):
            namespace["check_loaded_https"](front, {"version": "test-version"}, manifest)

    def test_https_headers_select_active_html_and_api_json(self):
        namespace = self.observer_namespace({})
        requests = []
        response = mock.Mock(status=200)
        response.getheader.return_value = "identity"
        response.read.return_value = b"response"
        fake_socket = mock.Mock()
        context = mock.Mock()
        context.wrap_socket.return_value = fake_socket
        with mock.patch("http.client.HTTPSConnection.request", side_effect=lambda *args, **kwargs: requests.append((args, kwargs))), \
                mock.patch("http.client.HTTPSConnection.getresponse", return_value=response), \
                mock.patch("socket.create_connection", return_value=mock.Mock()), \
                mock.patch("ssl.create_default_context", return_value=context):
            for path in ("/", "/assets/current.js", "/api/v1/settings/public"):
                namespace["https_fetch"]("gabugu.com", path)
        self.assertEqual([request[1]["headers"]["Accept"] for request in requests],
                         ["text/html", "*/*", "application/json"])
        self.assertTrue(all(request[1]["headers"]["Host"] == "gabugu.com" for request in requests))
        self.assertTrue(all("?release_sync=" in request[0][1] for request in requests))


if __name__ == "__main__":
    unittest.main()
