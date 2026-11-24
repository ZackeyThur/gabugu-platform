"""Local-only regressions for the frozen account-admin overlay observer."""

from __future__ import annotations

import base64
import contextlib
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import release_live as live
import release_sync_test as legacy


def overlay_record() -> dict:
    return {
        "role": "account-admin-overlay", "nginx_upstream": "gabugu_account_order",
        "route_methods": copy.deepcopy(live.OVERLAY_ROUTES), "method_maps": dict(live.OVERLAY_MAPS),
        "release_path": "/opt/gabugu/releases/overlay-test", "release_id": "overlay-test",
        "container_name": "overlay-test", "upstream_address": "127.0.0.1:18131",
        "version": "overlay-version", "expected_background_tasks": ["BillingProbe", "Scheduler"],
        "activation_mount_source": "/opt/gabugu/releases/overlay-test/activation",
        "activation_mount_destination": "/run/gabugu-release", "activation_marker": "activate.json",
    }


def nginx_config(root: str = "/expected") -> str:
    locations = "\n".join(
        "location = " + path + " { proxy_pass $" + live.OVERLAY_MAPS[method] + "; }"
        for method, paths in live.OVERLAY_ROUTES.items() for path in paths
    )
    maps = "\n".join(
        "map $request_method $" + name + " { default http://gabugu_active; "
        + method + " http://gabugu_account_order; }"
        for method, name in live.OVERLAY_MAPS.items()
    )
    return """http {
        upstream gabugu_active { server 127.0.0.1:18123; keepalive 64; }
        upstream gabugu_account_order { server 127.0.0.1:18131; keepalive 16; }
    """ + maps + """
        server { listen 80; server_name gabugu.com; location / { return 301 https://$host$request_uri; } }
        server { listen 443 ssl; server_name gabugu.com; root """ + root + ";" + locations + """
            location /api/ { proxy_pass http://gabugu_active; }
            location /v1/ { proxy_pass http://gabugu_active; }
            location /responses { proxy_pass http://gabugu_active; }
            location /v1beta/ { proxy_pass http://gabugu_active; }
            location /backend-api/ { proxy_pass http://gabugu_active; }
            location / { proxy_pass http://gabugu_active; }
        }
    }"""


class OverlayVerificationTest(unittest.TestCase):
    def namespace(self, baseline: dict | None = None) -> dict:
        return legacy.LiveVerificationTest().observer_namespace(baseline or {})

    def baseline(self) -> dict:
        base = legacy.LiveVerificationTest().expected()
        base["runtime"]["backend"].append(overlay_record())
        return base

    def test_two_backend_payload_is_allowlisted_and_legacy_shape_unchanged(self) -> None:
        for base, count in ((self.baseline(), 2), (legacy.LiveVerificationTest().expected(), 1)):
            base["runtime"]["backend"][-1]["secret"] = "must-not-copy"
            with self.subTest(count=count), mock.patch("release_live.subprocess.run", return_value=
                    subprocess.CompletedProcess([], 0, '{"runtime_live_verified":true}', "")) as run:
                live.observe_vps("vps", base)
                source = run.call_args.kwargs["input"]
                encoded = source.split('base64.b64decode("', 1)[1].split('"', 1)[0]
                payload = json.loads(base64.b64decode(encoded))
                self.assertEqual(len(payload["runtime"]["backend"]), count)
                self.assertNotIn("secret", payload["runtime"]["backend"][-1])
                if count == 2:
                    self.assertEqual(payload["runtime"]["backend"][1]["expected_background_tasks"],
                                     ["BillingProbe", "Scheduler"])

    def test_invalid_overlay_contract_never_reaches_ssh(self) -> None:
        changes = [
            {"role": "gateway"}, {"route_methods": {"GET": ["/v1/"]}},
            {"method_maps": {"GET": "arbitrary"}}, {"nginx_upstream": "arbitrary"},
            {"expected_background_tasks": []}, {"expected_background_tasks": ["a", "a"]},
            {"expected_background_tasks": "Scheduler"}, {"expected_background_tasks": ["bad task"]},
            {"release_id": "wrong"}, {"activation_marker": "../activate.json"},
            {"activation_mount_source": "/tmp"}, {"activation_mount_destination": "/run/other"},
            {"upstream_address": "127.0.0.1:18123"}, {"container_name": "backend-test"},
        ]
        with mock.patch("release_live.subprocess.run") as run:
            for change in changes:
                base = self.baseline(); base["runtime"]["backend"][1].update(change)
                with self.subTest(change=change), self.assertRaises(live.LiveVerificationError):
                    live.observe_vps("vps", base)
            base = self.baseline(); base["runtime"]["backend"].append(overlay_record())
            with self.assertRaises(live.LiveVerificationError):
                live.observe_vps("vps", base)
            run.assert_not_called()

    def test_finite_method_maps_and_all_primary_routes_pass(self) -> None:
        parse = self.namespace()["parse_nginx"]
        self.assertEqual(parse(nginx_config(), "gabugu.com", overlay_record()),
                         (["/expected"], ["127.0.0.1:18123", "127.0.0.1:18131"]))

    def test_single_backend_still_rejects_variable_proxy(self) -> None:
        namespace = self.namespace()
        with self.assertRaisesRegex(namespace["E"], "unexpected nginx proxy target"):
            namespace["parse_nginx"](nginx_config(), "gabugu.com")

    def test_frozen_nonrouting_marker_maps_do_not_expand_overlay_routes(self) -> None:
        namespace = self.namespace(); original = nginx_config()
        for method, name in live.OVERLAY_MAPS.items():
            marker = name.removesuffix("_backend") + "_marker"
            original += '\nmap $request_method $' + marker + ' { default ""; ' + method + ' "overlay-test"; }'
            original = original.replace("proxy_pass $" + name + ";", "add_header X-Gabugu-Account-Order $" + marker + " always; proxy_pass $" + name + ";")
        self.assertEqual(namespace["parse_nginx"](original, "gabugu.com", overlay_record())[1],
                         ["127.0.0.1:18123", "127.0.0.1:18131"])
        mutations = [
            original.replace('default ""; GET "overlay-test"', 'default "overlay-test"; GET "overlay-test"'),
            original.replace('GET "overlay-test"', 'POST "overlay-test"'),
            original.replace('GET "overlay-test"', 'GET "http://gabugu_account_order"'),
            original.replace('GET "overlay-test"', 'GET "$unknown"'),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { proxy_pass $gabugu_account_order_get_marker; }"),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { add_header X-Gabugu-Account-Order $gabugu_account_order_get_marker always; proxy_pass http://gabugu_active; }"),
            original.replace("add_header X-Gabugu-Account-Order $gabugu_account_order_get_marker always;", "add_header X-Gabugu-Account-Order $gabugu_account_order_put_marker always;", 1),
        ]
        for config in mutations:
            with self.subTest(config=config[-100:]), self.assertRaises(namespace["E"]):
                namespace["parse_nginx"](config, "gabugu.com", overlay_record())

    def test_standard_tls_and_mime_includes_are_content_verified(self) -> None:
        namespace = self.namespace()
        sections = {"/etc/letsencrypt/options-ssl-nginx.conf": "ssl_protocols TLSv1.2 TLSv1.3;\nssl_session_tickets off;",
                    "/etc/nginx/mime.types": "types { text/html html htm; application/javascript js; }"}
        namespace["nginx_include_source"] = lambda path: sections[path]
        config = nginx_config().replace("server_name gabugu.com; root", "server_name gabugu.com; include /etc/letsencrypt/options-ssl-nginx.conf; root")
        config = config.replace("location /api/", "location /brand/ { include /etc/nginx/mime.types; } location /api/")
        def dump() -> str:
            return config + "\n" + "\n".join("# configuration file " + path + ":\n" + value for path, value in sections.items())
        self.assertEqual(namespace["parse_nginx"](dump(), "gabugu.com", overlay_record())[1],
                         ["127.0.0.1:18123", "127.0.0.1:18131"])
        for path, injected in (
                ("/etc/letsencrypt/options-ssl-nginx.conf", "proxy_pass http://gabugu_active;"),
                ("/etc/letsencrypt/options-ssl-nginx.conf", "include /etc/nginx/mime.types;"),
                ("/etc/letsencrypt/options-ssl-nginx.conf", "rewrite ^ /v1/ last;"),
                ("/etc/nginx/mime.types", "root /wrong;"),
                ("/etc/nginx/mime.types", "location /v1/ { return 200; }")):
            original = sections[path]; sections[path] = original + "\n" + injected
            with self.subTest(injected=injected), self.assertRaises(namespace["E"]):
                namespace["parse_nginx"](dump(), "gabugu.com", overlay_record())
            sections[path] = original
        valid_dump = dump()
        sections["/etc/letsencrypt/options-ssl-nginx.conf"] += "\nreturn 403;"
        with self.assertRaisesRegex(namespace["E"], "dump differs"):
            namespace["parse_nginx"](valid_dump, "gabugu.com", overlay_record())

    def test_nginx_map_and_route_expansions_fail_closed(self) -> None:
        original = nginx_config()
        mutations = [
            original.replace("default http://gabugu_active", "default http://127.0.0.1:18123", 1),
            original.replace("GET http://gabugu_account_order", "POST http://gabugu_account_order", 1),
            original.replace("GET http://gabugu_account_order", "~GET http://gabugu_account_order", 1),
            original.replace("GET http://gabugu_account_order", "GET http://$arbitrary", 1),
            original.replace("GET http://gabugu_account_order;", "GET http://gabugu_account_order; POST http://gabugu_account_order;", 1),
            original.replace("GET http://gabugu_account_order;", "GET http://gabugu_account_order/extra;", 1),
            original.replace("map $request_method $gabugu_account_order_get_backend", "map $request_uri $gabugu_account_order_get_backend", 1),
            original.replace("location = /api/v1/admin/accounts {", "location /api/v1/admin/accounts {", 1),
            original.replace("location = /api/v1/admin/accounts {", "location = /api/v1/admin/accounts /extra {", 1),
            original.replace("location = /api/v1/admin/accounts {", "location = /api/v1/admin/accounts/ {", 1),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { proxy_pass http://gabugu_account_order; }"),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { proxy_pass $gabugu_account_order_get_backend; }"),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { proxy_pass http://127.0.0.1:18123; }"),
            original.replace("location /responses { proxy_pass http://gabugu_active; }", "location /responses { proxy_pass $unknown; }"),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { rewrite ^ /api/v1/admin/accounts last; proxy_pass http://gabugu_active; }"),
            original.replace("location /v1/ { proxy_pass http://gabugu_active; }", "location /v1/ { include /etc/nginx/unknown-routing.conf; proxy_pass http://gabugu_active; }"),
            original.replace("server_name gabugu.com; root", "server_name gabugu.com; rewrite ^/v1 /api/v1/admin/accounts last; root"),
            original.replace("proxy_pass $gabugu_account_order_get_backend;", "proxy_pass $gabugu_account_order_put_backend;", 1),
            original.replace("proxy_pass $gabugu_account_order_get_backend;", "proxy_pass $gabugu_account_order_get_backend; proxy_method GET;", 1),
            original.replace("proxy_pass $gabugu_account_order_get_backend;", "proxy_pass $gabugu_account_order_get_backend; rewrite ^ /v1/;", 1),
            original.replace("proxy_pass $gabugu_account_order_get_backend;", "proxy_pass $gabugu_account_order_get_backend; if ($x) { return 200; }", 1),
            original.replace("server 127.0.0.1:18131;", "server 127.0.0.1:18131; server 127.0.0.1:18132;", 1),
            original.replace("keepalive 16;", "keepalive 32;", 1),
            original.replace("server 127.0.0.1:18123;", "server 127.0.0.1:18123; server 127.0.0.1:18124;", 1),
            original.replace("server 127.0.0.1:18123;", "server 127.0.0.1:18123 down;", 1),
            original.replace("server_name gabugu.com; location /", "server_name gabugu.com; location /", 1)
                    .replace("return 301 https://$host$request_uri;", "proxy_pass http://gabugu_account_order;", 1),
            original + "map $request_method $other { default http://gabugu_account_order; }",
            original + "upstream alias_overlay { server 127.0.0.1:18131; }",
            original + "server { listen 80; server_name other.example; location / { proxy_pass http://127.0.0.1:18131; } }",
            original + "map $request_method $gabugu_account_order_get_backend { default http://gabugu_active; GET http://gabugu_account_order; }",
        ]
        namespace = self.namespace()
        for index, config in enumerate(mutations):
            with self.subTest(index=index), self.assertRaises(namespace["E"]):
                namespace["parse_nginx"](config, "gabugu.com", overlay_record())

    def dormant_status(self) -> dict:
        return {"configured": True, "release_id": "overlay-test", "registration_complete": True,
                "closed": False, "active": False, "registered": 2, "started": 0, "stopped": 0,
                "ready": False, "tasks": ["BillingProbe", "Scheduler"], "errors": {}}

    def docker(self, record: dict, pid: int = 123) -> dict:
        destination = record.get("activation_mount_destination", "/run/gabugu-release")
        return {"Image": "sha256:" + "a" * 64, "RestartCount": 0,
                "State": {"Pid": pid, "Health": {"Status": "healthy"}},
                "NetworkSettings": {"Ports": {"8080/tcp": [{"HostIp": "127.0.0.1",
                                                          "HostPort": record["upstream_address"].split(":")[1]}]}},
                "Mounts": [{"Destination": destination, "Source": record.get("activation_mount_source"),
                            "Type": "bind", "RW": False}],
                "Config": {"Env": ["GABUGU_RELEASE_ID=" + record.get("release_id", ""),
                                   "GABUGU_BACKGROUND_ACTIVATION_FILE=" + destination + "/activate.json"]}}

    def test_standby_allows_ready_false_and_rejects_extra_background_or_mutable_marker(self) -> None:
        namespace = self.namespace(); record = overlay_record(); docker = self.docker(record)
        namespace["safe_root"] = lambda p: p
        status = self.dormant_status()
        namespace["backend_json"] = lambda *args: status
        check = namespace["check_overlay_standby"]
        self.assertEqual(check(record, docker, 123), status)
        for change in ({"active": True}, {"ready": True}, {"started": 1}, {"started": False},
                       {"stopped": 1}, {"registration_complete": False}, {"closed": True},
                       {"configured": False}, {"release_id": "wrong"}, {"registered": 3},
                       {"tasks": ["BillingProbe", "Scheduler", "UnexpectedWriter"]},
                       {"tasks": ["BillingProbe", "BillingProbe"]}, {"errors": {"Scheduler": "failed"}}):
            status = self.dormant_status(); status.update(change)
            with self.subTest(change=change), self.assertRaises(namespace["E"]):
                check(record, docker, 123)
        status = self.dormant_status()
        for mutate in (lambda d: d["Mounts"][0].update(RW=True),
                       lambda d: d["Mounts"][0].update(Source="/wrong"),
                       lambda d: d["Mounts"].append({"Destination": "/run/gabugu-release/activate.json", "RW": True}),
                       lambda d: d["Config"]["Env"].append("GABUGU_RELEASE_ID=other")):
            changed = copy.deepcopy(docker); mutate(changed)
            with self.assertRaises(namespace["E"]): check(record, changed, 123)
        with mock.patch("os.path.lexists", return_value=True), self.assertRaisesRegex(namespace["E"], "marker exists"):
            check(record, docker, 123)

    def test_two_backend_main_checks_both_identities_and_detects_second_instance_drift(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder); base = self.baseline(); records = base["runtime"]["backend"]
            for record in records:
                release = root / record["container_name"]; source = release / "source"; source.mkdir(parents=True)
                (source / "file").write_bytes(record["container_name"].encode())
                manifest = source / "source-manifest.json"
                manifest.write_text(json.dumps([{"path": "file", "sha256": hashlib.sha256((source / "file").read_bytes()).hexdigest()}]))
                record.update(release_path=str(release), source_manifest_sha256=hashlib.sha256(manifest.read_bytes()).hexdigest(),
                              binary_sha256="b" * 64, image_digest="sha256:" + "a" * 64)
            records[1]["activation_mount_source"] = str(root / "activation")
            (root / "activation").mkdir()
            front = base["runtime"]["frontend"]; dist = root / "dist"; dist.mkdir(); (dist / "assets").mkdir()
            (dist / "index.html").write_bytes(b"index")
            (dist / "assets/current.js").write_bytes(b"current")
            manifest = {"source_archive_sha256": "a" * 64, "assets": [
                {"path": path, "sha256": hashlib.sha256((dist / path).read_bytes()).hexdigest()}
                for path in ("index.html", "assets/current.js")]}
            front_dir = root / "front"; front_dir.mkdir(); fm = front_dir / "manifest.json"; fm.write_text(json.dumps(manifest))
            front.update(release_path=str(front_dir), nginx_root=str(dist), archive_sha256="a" * 64,
                         dist_manifest_sha256=hashlib.sha256(fm.read_bytes()).hexdigest(),
                         source_manifest_sha256=hashlib.sha256(fm.read_bytes()).hexdigest())
            namespace = self.namespace(base); namespace["safe_root"] = lambda p: p
            original_hash = namespace["hfile"]
            bad_binary = False
            namespace["hfile"] = lambda p: ("c" if bad_binary and p == "/proc/124/exe" else "b") * 64 if p.startswith("/proc/") else original_hash(p)
            dockers = {record["container_name"]: self.docker(record, 123 + i) for i, record in enumerate(records)}
            namespace["cmd"] = lambda *args: json.dumps(dockers[args[-1]]) if args[0] == "docker" else nginx_config(str(dist))
            namespace["https_fetch"] = lambda host, path: (b'<script src="/assets/current.js"></script>' if path == "/" else
                b'{"data":{"version":"test-version"}}' if path == "/api/v1/settings/public" else
                b'{"status":"ok"}' if path == "/health" else b"current")
            def backend_json(address: str, path: str) -> dict:
                if path == "/health/rolling": return self.dormant_status()
                if path == "/health": return {"status": "ok"}
                record = next(item for item in records if item["upstream_address"] == address)
                return {"data": {"version": record["version"]}}
            namespace["backend_json"] = backend_json
            realpath = os.path.realpath
            with mock.patch("os.path.realpath", side_effect=lambda p: records[0]["release_path"] if p == "/opt/gabugu/current-release" else realpath(p)):
                with contextlib.redirect_stdout(io.StringIO()) as output: namespace["main"]()
                result = json.loads(output.getvalue())
                self.assertTrue(result["runtime_live_verified"])
                self.assertEqual(len(result["backends"]), 2)
                self.assertFalse(result["backends"][1]["background"]["ready"])
                bad_binary = True
                with self.assertRaisesRegex(namespace["E"], "running binary digest differs"): namespace["main"]()
                bad_binary = False
                dockers[records[1]["container_name"]]["Image"] = "sha256:" + "c" * 64
                with self.assertRaisesRegex(namespace["E"], "container image digest differs"): namespace["main"]()
                original_docker = self.docker(records[1], 124)
                for mutate, message in (
                        (lambda d: d["State"].update(Pid=0), "running PID"),
                        (lambda d: d["State"]["Health"].update(Status="unhealthy"), "not healthy"),
                        (lambda d: d["NetworkSettings"]["Ports"]["8080/tcp"][0].update(HostPort="18132"), "container port differs"),
                        (lambda d: d["NetworkSettings"]["Ports"]["8080/tcp"].append({"HostIp": "0.0.0.0", "HostPort": "18131"}), "unexpected backend port")):
                    changed = copy.deepcopy(original_docker); mutate(changed)
                    dockers[records[1]["container_name"]] = changed
                    with self.subTest(message=message), self.assertRaisesRegex(namespace["E"], message):
                        namespace["main"]()
                dockers[records[1]["container_name"]] = original_docker
                overlay_source = Path(records[1]["release_path"]) / "source/file"
                overlay_source.write_bytes(b"tampered")
                with self.assertRaisesRegex(namespace["E"], "backend source digest mismatch"): namespace["main"]()
                overlay_source.write_bytes(records[1]["container_name"].encode())
                original_http = namespace["backend_json"]
                namespace["backend_json"] = lambda address, path: {"data": {"version": "wrong"}} if (
                    address == records[1]["upstream_address"] and path == "/api/v1/settings/public") else original_http(address, path)
                with self.assertRaisesRegex(namespace["E"], "backend health version differs"): namespace["main"]()


if __name__ == "__main__":
    unittest.main()
