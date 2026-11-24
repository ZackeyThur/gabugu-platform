#!/usr/bin/env python3
"""Read-only production observer executed on the VPS through SSH.

The module also exposes ``observe_vps`` for the local verifier. The observer
does not deploy, restart, mutate files, or print configuration secrets.
"""

from __future__ import annotations

import base64
import json
import re
import subprocess
from typing import Any


ALIAS_RE = re.compile(r"^[A-Za-z0-9_.-]+$")
OVERLAY_ROUTES = {
    "GET": ["/api/v1/admin/accounts", "/api/v1/admin/accounts/upstream-billing-rates",
            "/api/v1/admin/accounts/data"],
    "PUT": ["/api/v1/admin/accounts/reorder"],
}
OVERLAY_MAPS = {"GET": "gabugu_account_order_get_backend",
                "PUT": "gabugu_account_order_put_backend"}


class LiveVerificationError(Exception):
    pass


def require(condition: bool, message: str) -> None:
    if not condition:
        raise LiveVerificationError(message)


def observer_source() -> str:
    """Return the self-contained script run as ``python3 -`` on the VPS."""
    return _OBSERVER_SOURCE


def observe_vps(alias: str, baseline: dict[str, Any], timeout: int = 600) -> dict[str, Any]:
    require(isinstance(alias, str) and ALIAS_RE.fullmatch(alias) is not None,
            "live-vps must be a simple SSH alias")
    require(not alias.startswith("-"), "live-vps must not be an SSH option")
    validate_expected(baseline)
    fields = ("release_path", "container_name", "upstream_address", "source_manifest_sha256",
              "binary_sha256", "image_digest", "nginx_root", "manifest_filename", "hostname",
              "dist_manifest_sha256", "archive_sha256", "version", "index_asset", "role",
              "nginx_upstream", "route_methods", "method_maps", "release_id",
              "expected_background_tasks", "activation_mount_source", "activation_mount_destination",
              "activation_marker")
    runtime = baseline["runtime"]
    payload_data = {"runtime": {"backend": [{key: value for key, value in record.items() if key in fields}
                                           for record in runtime["backend"]],
                                "frontend": {key: value for key, value in runtime["frontend"].items()
                                             if key in fields}}}
    payload = base64.b64encode(json.dumps(payload_data, separators=(",", ":")).encode()).decode("ascii")
    script = observer_source().replace("__PAYLOAD__", payload)
    try:
        result = subprocess.run(["ssh", alias, "python3", "-"], input=script,
                                text=True, capture_output=True, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise LiveVerificationError("SSH observer unavailable or timed out") from None
    try:
        observed = json.loads(result.stdout)
    except json.JSONDecodeError:
        raise LiveVerificationError("VPS observer returned invalid JSON") from None
    require(isinstance(observed, dict), "VPS observer returned invalid result")
    if result.returncode != 0 or observed.get("runtime_live_verified") is not True:
        reason = observed.get("mismatch", "unconfirmed state")
        if not isinstance(reason, str) or len(reason) > 200 or not reason.isprintable():
            reason = "unconfirmed state"
        raise LiveVerificationError("VPS observed state differs from expected baseline: " + reason)
    return observed


def validate_expected(baseline: dict[str, Any]) -> None:
    runtime = baseline.get("runtime", {})
    backends = runtime.get("backend", [])
    require(isinstance(backends, list) and len(backends) in (1, 2),
            "live observer requires one backend or a restricted account-admin overlay")
    front = runtime.get("frontend", {})
    for record in (*backends, front):
        require(isinstance(record, dict), "unexpected live runtime record")
        require(re.fullmatch(r"/opt/gabugu/releases/[A-Za-z0-9_.-]+", record.get("release_path", "")) is not None
                and record["release_path"].split("/")[-1] not in (".", ".."), "unexpected live release path")
    for record in backends:
        require(re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]*", record.get("container_name", "")) is not None,
                "unexpected container name")
        require(re.fullmatch(r"127\.0\.0\.1:[0-9]{1,5}", record.get("upstream_address", "")) is not None,
                "unexpected backend upstream address")
        require(isinstance(record.get("version"), str) and record["version"],
                "expected backend version missing")
    if len(backends) == 2:
        validate_overlay(backends)
    require(re.fullmatch(r"/var/www/gabugu-frontend/[A-Za-z0-9_.-]+", front.get("nginx_root", "")) is not None
            and front["nginx_root"].split("/")[-1] not in (".", ".."), "unexpected frontend root")
    require(front.get("manifest_filename") == "manifest.json", "unexpected frontend manifest filename")
    require(re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]*", front.get("hostname", "")) is not None,
            "unexpected frontend hostname")
    require(re.fullmatch(r"/assets/[A-Za-z0-9_.-]+\.js", front.get("index_asset", "")) is not None,
            "unexpected current frontend entrypoint")


def validate_overlay(backends: list[dict[str, Any]]) -> None:
    """Allow only the frozen account-admin method/path and dormant-worker contract."""
    main, overlay = backends
    require(main.get("role") in (None, "primary"), "unexpected primary backend role")
    require(overlay.get("role") == "account-admin-overlay", "unexpected second backend role")
    require(overlay.get("nginx_upstream") == "gabugu_account_order"
            and overlay.get("route_methods") == OVERLAY_ROUTES
            and overlay.get("method_maps") == OVERLAY_MAPS, "unexpected overlay route contract")
    for field in ("container_name", "upstream_address", "release_path"):
        require(main[field] != overlay[field], "overlay must have a distinct " + field)
    require(overlay.get("release_id") == overlay["release_path"].rsplit("/", 1)[1],
            "overlay release ID differs from source release")
    tasks = overlay.get("expected_background_tasks")
    require(isinstance(tasks, list) and bool(tasks)
            and all(isinstance(task, str) and re.fullmatch(r"[A-Za-z][A-Za-z0-9_.:-]*", task) for task in tasks)
            and len(set(tasks)) == len(tasks), "overlay expected background tasks must be exact and unique")
    require(overlay.get("activation_mount_source") == overlay["release_path"] + "/activation"
            and overlay.get("activation_mount_destination") == "/run/gabugu-release"
            and overlay.get("activation_marker") == "activate.json", "unexpected overlay activation contract")


_OBSERVER_SOURCE = r'''import base64, hashlib, http.client, json, os, re, socket, ssl, stat, subprocess, sys, time, urllib.parse

PAYLOAD = json.loads(base64.b64decode("__PAYLOAD__"))
IMAGE = re.compile(r"sha256:[0-9a-f]{64}\Z")

class E(Exception): pass
def req(c, m):
    if not c: raise E(m)
def hfile(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for b in iter(lambda: f.read(1048576), b""): h.update(b)
    return h.hexdigest()
def rel_path(root, rel):
    rel = rel_path_value(rel)
    p = os.path.join(root, *rel.split("/"))
    cur = root
    for part in rel.split("/"):
        cur = os.path.join(cur, part)
        st = os.lstat(cur)
        req(not stat.S_ISLNK(st.st_mode), "symlink or reparse point: " + rel)
    req(os.path.commonpath([os.path.realpath(root), os.path.realpath(p)]) == os.path.realpath(root), "path escaped root")
    return p
def rel_path_value(value):
    req(isinstance(value, str) and value and not value.startswith("/") and "\\" not in value
        and not any(ord(c) < 32 for c in value), "unsafe manifest path")
    parts = value.split("/")
    req(all(x and x not in (".", "..") for x in parts), "unsafe manifest path")
    return value
def safe_root(path):
    cur = "/"
    for part in path.strip("/").split("/"):
        cur = os.path.join(cur, part)
        req(not stat.S_ISLNK(os.lstat(cur).st_mode), "expected root contains symlink")
    req(os.path.isdir(path), "expected root is unavailable")
    return path
def cmd(*args):
    p = subprocess.run(args, capture_output=True, text=True, timeout=30, check=False)
    req(p.returncode == 0, "remote command failed")
    return p.stdout
def read_json(path):
    with open(path, encoding="utf-8") as f: return json.load(f)
def check_manifest(root, manifest, expected, label):
    safe_root(root)
    req(not stat.S_ISLNK(os.lstat(manifest).st_mode), label + " manifest is symlink")
    req(hfile(manifest) == expected.lower(), label + " manifest digest mismatch")
    data = read_json(manifest)
    req(isinstance(data, list) and data, label + " source manifest must be nonempty array")
    seen = set()
    for item in data:
        req(isinstance(item, dict) and isinstance(item.get("path"), str), label + " malformed entry")
        rel = rel_path_value(item["path"])
        key = rel.casefold(); req(key not in seen, label + " duplicate path"); seen.add(key)
        expected_sha = item.get("sha256", "").lower(); req(re.fullmatch(r"[0-9a-f]{64}", expected_sha) is not None, label + " malformed hash")
        actual = rel_path(root, rel); req(stat.S_ISREG(os.stat(actual).st_mode), label + " nonregular file")
        req(hfile(actual) == expected_sha, label + " source digest mismatch")
    actual_set = set()
    for directory, dirs, files in os.walk(root, followlinks=False):
        for name in dirs + files:
            p = os.path.join(directory, name); req(not os.path.islink(p), label + " symlink in source")
        for name in files:
            actual_set.add(os.path.relpath(os.path.join(directory, name), root).replace(os.sep, "/"))
    metadata = {name for name in ("source-manifest.json", "frontend-dist-manifest.json") if name in actual_set}
    req(actual_set == {item["path"] for item in data} | metadata, label + " file set differs")
    return len(data)
def parse_nginx(text, hostname, overlay=None):
    # nginx -T concatenates included config files; named upstreams are resolved
    # from the same expanded dump. Extract only routing data for this hostname.
    tokens = re.findall(r'"[^"\\]*(?:\\.[^"\\]*)*"|\x27[^\x27]*\x27|[{};]|[^\s{};]+', re.sub(r"(?m)#.*$", "", text))
    stack = []; directive = []; blocks = []; upstreams = {}; servers = []; all_blocks = []
    for token in tokens:
        if token == "{":
            block = {"head": directive, "directives": [], "children": [],
                     "parent_head": stack[-1]["head"] if stack else []}; directive = []
            all_blocks.append(block)
            if stack: stack[-1]["children"].append(block)
            else: blocks.append(block)
            stack.append(block)
        elif token == "}":
            req(bool(stack) and not directive, "nginx block mismatch"); stack.pop(); directive = []
        elif token == ";":
            if stack: stack[-1]["directives"].append(directive)
            directive = []
        else: directive.append(token.strip('"\x27'))
    req(not stack and not directive, "nginx block mismatch")
    def walk(block):
        head = block["head"]; ds = block["directives"]
        if len(head) == 2 and head[0] == "upstream":
            if overlay: req(head[1] not in upstreams, "duplicate nginx upstream")
            upstreams[head[1]] = [d[1] for d in ds if len(d) >= 2 and d[0] == "server"]
        tls = any(d and d[0] == "listen" and len(d) > 1 and d[1].split(":")[-1] == "443" and "ssl" in d[2:] for d in ds)
        if head == ["server"] and tls and any(d and d[0] == "server_name" and hostname in d[1:] for d in ds):
            servers.append(block)
        for child in block["children"]: walk(child)
    for block in blocks: walk(block)
    req(bool(servers), "nginx hostname server missing")
    verified_includes = check_nginx_includes(text, servers) if overlay else set()
    variable_targets = check_overlay_routes(all_blocks, servers, upstreams, overlay, verified_includes) if overlay else {}
    roots = []; proxies = []
    def routes(block):
        for d in block["directives"]:
            if len(d) == 2 and d[0] == "root": roots.append(d[1])
            if len(d) == 2 and d[0] == "proxy_pass": proxies.append(d[1])
        for child in block["children"]: routes(child)
    for server in servers: routes(server)
    addresses = []
    for proxy in proxies:
        targets = variable_targets.get(proxy, [proxy])
        for value in targets:
            match = re.fullmatch(r"https?://([A-Za-z0-9_.:-]+)(?:/.*)?", value)
            req(match is not None, "unexpected nginx proxy target")
            target = match.group(1); addresses.extend(upstreams.get(target, [target]))
    return sorted(set(roots)), sorted(set(addresses))
def nginx_include_source(path):
    with open(path, encoding="utf-8") as handle: value = handle.read(131073)
    req(len(value) <= 131072, "nginx standard include exceeded size limit")
    return value
def check_nginx_includes(text, servers):
    allowed = {"/etc/letsencrypt/options-ssl-nginx.conf", "/etc/nginx/mime.types"}
    used = set()
    def visit(block):
        for d in block["directives"]:
            if d and d[0] == "include":
                req(len(d) == 2 and d[1] in allowed, "unknown nginx server or location include")
                used.add(d[1])
        for child in block["children"]: visit(child)
    for server in servers: visit(server)
    headers = list(re.finditer(r"(?m)^# configuration file ([^\r\n]+):[ \t]*\r?$", text))
    for path in used:
        matches = [i for i, header in enumerate(headers) if header.group(1) == path]
        req(len(matches) == 1, "nginx standard include dump missing or duplicated")
        index = matches[0]; start = headers[index].end()
        end = headers[index + 1].start() if index + 1 < len(headers) else len(text)
        body = text[start:end].strip()
        # Compare the actual file too: an embedded fake dump header must not
        # hide trailing routing directives from section validation.
        req(nginx_include_source(path).strip() == body, "nginx standard include changed or dump differs")
        tokens = re.findall(r'"[^"\\]*(?:\\.[^"\\]*)*"|\x27[^\x27]*\x27|[{};]|[^\s{};]+', re.sub(r"(?m)#.*$", "", body))
        if path.endswith("mime.types"):
            req(len(tokens) >= 3 and tokens[:2] == ["types", "{"] and tokens[-1] == "}", "nginx MIME include shape differs")
            tokens = tokens[2:-1]
        statement = []
        ssl = {"ssl_session_cache", "ssl_session_timeout", "ssl_session_tickets", "ssl_protocols",
               "ssl_prefer_server_ciphers", "ssl_ciphers", "ssl_ecdh_curve", "ssl_dhparam"}
        for token in tokens:
            req(token not in ("{", "}"), "nginx standard include has nested block")
            if token == ";":
                req(len(statement) >= 2 and (re.fullmatch(r"[A-Za-z0-9.+_-]+/[A-Za-z0-9.+_-]+", statement[0]) is not None
                    if path.endswith("mime.types") else statement[0] in ssl), "nginx standard include has unverified directive")
                statement = []
            else: statement.append(token)
        req(not statement, "nginx standard include has unterminated directive")
    return used
def check_overlay_routes(blocks, servers, upstreams, overlay, verified_includes):
    # Resolve only these two frozen, finite method maps. No variable expansion,
    # regex map keys, nested maps, URL suffixes, or fallback IP literals are valid.
    route_methods = {"GET": ["/api/v1/admin/accounts", "/api/v1/admin/accounts/upstream-billing-rates", "/api/v1/admin/accounts/data"],
                     "PUT": ["/api/v1/admin/accounts/reorder"]}
    names = {"GET": "gabugu_account_order_get_backend", "PUT": "gabugu_account_order_put_backend"}
    req(overlay.get("nginx_upstream") == "gabugu_account_order" and overlay.get("route_methods") == route_methods
        and overlay.get("method_maps") == names, "unexpected overlay route contract")
    req(upstreams.get("gabugu_account_order") == [overlay["upstream_address"]], "overlay nginx upstream differs")
    req(len(upstreams.get("gabugu_active", [])) == 1, "primary nginx upstream must be singular")
    overlay_upstream = [b for b in blocks if b["head"] == ["upstream", "gabugu_account_order"]]
    req(len(overlay_upstream) == 1 and overlay_upstream[0]["parent_head"] in ([], ["http"])
        and not overlay_upstream[0]["children"] and sorted(overlay_upstream[0]["directives"]) == sorted([
            ["server", overlay["upstream_address"]], ["keepalive", "16"]]), "overlay nginx upstream shape differs")
    primary_upstream = [b for b in blocks if b["head"] == ["upstream", "gabugu_active"]]
    req(len(primary_upstream) == 1 and not primary_upstream[0]["children"]
        and [d for d in primary_upstream[0]["directives"] if d and d[0] == "server"] == [
            ["server", upstreams["gabugu_active"][0]]], "primary nginx upstream shape differs")
    variables = {"$" + name for name in names.values()}
    allowed_maps = []; allowed_locations = []; targets = {}; marker_headers = []
    marker_names = {method: name.removesuffix("_backend") + "_marker" for method, name in names.items()}
    marker_maps = [b for b in blocks if any("$" + name in b["head"] for name in marker_names.values())]
    if marker_maps:
        req(len(marker_maps) == 2, "overlay marker maps missing or duplicated")
        for method, name in marker_names.items():
            matches = [b for b in marker_maps if b["head"] == ["map", "$request_method", "$" + name]]
            req(len(matches) == 1, "overlay marker method map differs")
            block = matches[0]
            req(isinstance(overlay.get("release_id"), str) and block["parent_head"] in ([], ["http"])
                and not block["children"] and sorted(block["directives"]) == sorted([
                    ["default", ""], [method, overlay["release_id"]]]), "overlay marker map values differ")
            allowed_maps.append(block)
    for method, name in names.items():
        matches = [b for b in blocks if b["head"] == ["map", "$request_method", "$" + name]]
        req(len(matches) == 1, "overlay nginx method map missing or duplicated")
        block = matches[0]
        req(block["parent_head"] in ([], ["http"]) and not block["children"] and sorted(block["directives"]) == sorted([
            ["default", "http://gabugu_active"], [method, "http://gabugu_account_order"]]), "overlay nginx method map differs")
        allowed_maps.append(block)
        targets["$" + name] = ["http://gabugu_active", "http://gabugu_account_order"]
    for server in servers:
        for method, paths in route_methods.items():
            for path in paths:
                matches = [b for b in server["children"] if b["head"] == ["location", "=", path]]
                req(len(matches) == 1, "overlay exact nginx location missing or duplicated")
                block = matches[0]; ds = block["directives"]
                req(not block["children"] and [d for d in ds if d and d[0] == "proxy_pass"] == [
                    ["proxy_pass", "$" + names[method]]], "overlay exact nginx location proxy differs")
                req(not any(d and d[0] in {"rewrite", "return", "try_files", "error_page", "proxy_method", "set", "include"}
                            for d in ds), "overlay exact nginx location has routing override")
                allowed_locations.append(block)
                header = ["add_header", "X-Gabugu-Account-Order", "$" + marker_names[method], "always"]
                if marker_maps:
                    req([d for d in ds if d and d[0] == "add_header" and len(d) > 1
                         and d[1].lower() == "x-gabugu-account-order"] == [header], "overlay exact marker header differs")
                    marker_headers.append((block, header))
    for block in blocks:
        if any(block is allowed for allowed in allowed_maps): continue
        for directive in block["directives"]:
            if block is overlay_upstream[0] and directive == ["server", overlay["upstream_address"]]: continue
            mentions_overlay = any(value in variables or "gabugu_account_order" in value
                                   or overlay["upstream_address"] in value for value in directive)
            if mentions_overlay:
                req((any(block is allowed for allowed in allowed_locations)
                     and len(directive) == 2 and directive[0] == "proxy_pass" and directive[1] in variables)
                    or any(block is allowed and directive == header for allowed, header in marker_headers),
                    "overlay target referenced outside exact account locations")
        if block["head"][:1] == ["map"]:
            req(not any(value in variables for value in block["head"]), "overlay map redefined")
    def main_routes(block):
        if any(block is allowed for allowed in allowed_locations): return
        for d in block["directives"]:
            req(not d or d[0] not in {"rewrite", "proxy_method"}, "non-overlay nginx route has unverified routing override")
            if d and d[0] == "include":
                req(len(d) == 2 and d[1] in verified_includes, "non-overlay nginx route has unverified include")
            if d and d[0] == "proxy_pass":
                req(d == ["proxy_pass", "http://gabugu_active"], "non-overlay nginx route must use named primary upstream")
        for child in block["children"]: main_routes(child)
    for server in servers: main_routes(server)
    return targets
_SSL_CONTEXT = None
def https_fetch(hostname, path):
    global _SSL_CONTEXT
    class LoopbackTLS(http.client.HTTPSConnection):
        def connect(self):
            raw = socket.create_connection(("127.0.0.1", 443), timeout=self.timeout)
            try:
                self.sock = self._context.wrap_socket(raw, server_hostname=self.host)
            except Exception:
                raw.close(); raise
    if _SSL_CONTEXT is None:
        _SSL_CONTEXT = ssl.create_default_context()
    connection = LoopbackTLS(hostname, timeout=8, context=_SSL_CONTEXT)
    try:
        query = "release_sync=" + str(time.time_ns())
        uri = urllib.parse.quote(path, safe="/.-_") + "?" + query
        # Production selects the independently released SPA for HTML requests;
        # omitting Accept can select the backend's historical embedded index.
        accept = "text/html" if path == "/" else "application/json" if path.startswith("/api/") else "*/*"
        connection.request("GET", uri, headers={"Host": hostname, "Accept": accept, "Accept-Encoding": "identity", "Cache-Control": "no-cache"})
        response = connection.getresponse()
        req(response.status == 200, "loaded HTTPS route returned non-200")
        req(response.getheader("Content-Encoding", "identity") in ("identity", ""), "unexpected HTTPS content encoding")
        body = response.read(32 * 1024 * 1024 + 1)
        req(len(body) <= 32 * 1024 * 1024, "HTTPS response exceeded size limit")
        return body
    finally:
        connection.close()
def check_loaded_https(front, backend, manifest):
    count = 0
    req(any(item["path"] == front["index_asset"].lstrip("/") for item in manifest["assets"]), "entrypoint absent from asset manifest")
    for item in manifest["assets"]:
        if item["path"] == "index.html": continue
        body = https_fetch(front["hostname"], "/" + item["path"])
        req(hashlib.sha256(body).hexdigest() == item["sha256"].lower(), "loaded HTTPS asset digest differs")
        count += 1
    index = https_fetch(front["hostname"], "/").decode("utf-8")
    script_sources = re.findall(r'<script\b[^>]*\bsrc=["\x27]([^"\x27]+)["\x27]', index, re.I)
    req(front["index_asset"] in script_sources, "loaded HTTPS index entrypoint differs")
    response = json.loads(https_fetch(front["hostname"], "/api/v1/settings/public"))
    req(isinstance(response, dict) and isinstance(response.get("data"), dict) and response["data"].get("version") == backend["version"], "loaded HTTPS backend version differs")
    return count
def backend_json(address, path):
    host, port = address.split(":")
    req(host == "127.0.0.1" and port.isdecimal() and 0 < int(port) < 65536, "unsafe backend health address")
    connection = http.client.HTTPConnection(host, int(port), timeout=8)
    try:
        connection.request("GET", path, headers={"Accept": "application/json", "Accept-Encoding": "identity", "Cache-Control": "no-cache"})
        response = connection.getresponse()
        req(response.status == 200 and response.getheader("Content-Encoding", "identity") in ("identity", ""), "backend health response differs")
        body = response.read(1024 * 1024 + 1)
        req(len(body) <= 1024 * 1024, "backend health response exceeded size limit")
        value = json.loads(body); req(isinstance(value, dict), "backend health response is not object")
        return value
    finally:
        connection.close()
def check_backend_http(back):
    health = backend_json(back["upstream_address"], "/health")
    req(health.get("status") == "ok", "backend health status differs")
    public = backend_json(back["upstream_address"], "/api/v1/settings/public")
    req(isinstance(public.get("data"), dict) and public["data"].get("version") == back["version"], "backend health version differs")
def check_overlay_standby(back, docker, pid):
    status = backend_json(back["upstream_address"], "/health/rolling")
    req(status.get("configured") is True and status.get("release_id") == back["release_id"], "overlay gate release binding differs")
    req(status.get("registration_complete") is True and status.get("closed") is False, "overlay background registry is not sealed")
    tasks = status.get("tasks")
    req(isinstance(tasks, list) and all(isinstance(task, str) for task in tasks)
        and sorted(tasks) == sorted(back["expected_background_tasks"])
        and type(status.get("registered")) is int and status["registered"] == len(tasks), "overlay background task inventory differs")
    req(status.get("active") is False and status.get("ready") is False
        and type(status.get("started")) is int and status["started"] == 0
        and type(status.get("stopped")) is int and status["stopped"] == 0
        and status.get("errors", {}) == {}, "overlay background workers are not dormant")
    destination = back["activation_mount_destination"]; source = back["activation_mount_source"]
    mounts = [item for item in docker.get("Mounts", []) if item.get("Destination") == destination]
    req(len(mounts) == 1 and mounts[0].get("Source") == source and mounts[0].get("RW") is False
        and mounts[0].get("Type") == "bind", "overlay activation mount is not the expected read-only bind")
    # A parent or child writable mount could mask the read-only marker directory.
    for item in docker.get("Mounts", []):
        target = item.get("Destination", "")
        req(not (target != destination and (target.startswith(destination + "/") or destination.startswith(target.rstrip("/") + "/"))),
            "overlay activation mount is masked")
    env = docker.get("Config", {}).get("Env", [])
    for name, value in {"GABUGU_BACKGROUND_ACTIVATION_FILE": destination + "/" + back["activation_marker"],
                        "GABUGU_RELEASE_ID": back["release_id"]}.items():
        req([entry for entry in env if entry.startswith(name + "=")] == [name + "=" + value], "overlay gate environment differs")
    safe_root(source)
    req(not os.path.lexists(os.path.join(source, back["activation_marker"]))
        and not os.path.lexists("/proc/%d/root" % pid + destination + "/" + back["activation_marker"]), "overlay activation marker exists")
    return status
def inspect_backend(back, strict=False):
    release = back["release_path"]; safe_root(release)
    source = os.path.join(release, "source"); source_manifest = os.path.join(source, "source-manifest.json")
    if not os.path.isfile(source_manifest): source_manifest = os.path.join(release, "source-manifest.json")
    count = check_manifest(source, source_manifest, back["source_manifest_sha256"], "backend")
    docker = json.loads(cmd("docker", "inspect", "--format", "{{json .}}", back["container_name"]))
    image = docker.get("Image"); pid = int(docker.get("State", {}).get("Pid", 0)); health = docker.get("State", {}).get("Health", {}).get("Status")
    req(IMAGE.fullmatch(image or "") is not None and image == back["image_digest"], "container image digest differs")
    req(health == "healthy", "container is not healthy")
    req(pid > 0, "container has no running PID")
    binary_sha = hfile("/proc/%d/exe" % pid)
    req(binary_sha == back["binary_sha256"], "running binary digest differs")
    port_bindings = docker.get("NetworkSettings", {}).get("Ports", {}).get("8080/tcp", []) or []
    req(any(p.get("HostIp", "") + ":" + p.get("HostPort", "") == back["upstream_address"] for p in port_bindings), "container port differs from expected upstream")
    if strict:
        req(port_bindings == [{"HostIp": "127.0.0.1", "HostPort": back["upstream_address"].split(":")[1]}], "container has unexpected backend port bindings")
        check_backend_http(back)
    report = {"release_path": release, "source_files": count, "container_image": image, "pid": pid,
              "health": health, "restart_count": docker.get("RestartCount"), "running_binary_sha256": binary_sha, "https_version": back["version"]}
    if back.get("role") == "account-admin-overlay": report["background"] = check_overlay_standby(back, docker, pid)
    return docker, report
def main():
    base = PAYLOAD; runtime = base["runtime"]
    backends = runtime["backend"]; back = backends[0]; release = back["release_path"]; current = "/opt/gabugu/current-release"
    overlay = backends[1] if len(backends) == 2 else None
    req(os.path.realpath(current) == release, "current-release differs from expected release")
    observations = [inspect_backend(record, bool(overlay)) for record in backends]
    front = runtime["frontend"]; safe_root(front["release_path"]); safe_root(front["nginx_root"])
    fmanifest = rel_path(front["release_path"], front["manifest_filename"])
    req(hfile(fmanifest) == front["dist_manifest_sha256"] == front["source_manifest_sha256"], "frontend manifest digest differs")
    fm = read_json(fmanifest); req(isinstance(fm, dict) and isinstance(fm.get("assets"), list), "frontend manifest malformed")
    req(fm.get("source_archive_sha256") == front["archive_sha256"], "frontend source archive differs")
    assets = 0; seen = set()
    req(bool(fm["assets"]), "frontend assets empty")
    for item in fm["assets"]:
        req(isinstance(item, dict), "frontend asset malformed"); rel = rel_path_value(item.get("path")); p = rel_path(front["nginx_root"], rel)
        req(rel.casefold() not in seen, "frontend duplicate asset"); seen.add(rel.casefold())
        req(stat.S_ISREG(os.stat(p).st_mode), "frontend nonregular asset")
        req(hfile(p) == item.get("sha256", "").lower(), "frontend asset digest mismatch"); assets += 1
    nginx = cmd("nginx", "-T")
    roots, addresses = parse_nginx(nginx, front["hostname"], overlay)
    req(roots == [front["nginx_root"]], "nginx active root differs")
    req(addresses == sorted(record["upstream_address"] for record in backends), "nginx active upstream differs")
    https_assets = check_loaded_https(front, back, fm)
    if overlay: req(json.loads(https_fetch(front["hostname"], "/health")).get("status") == "ok", "loaded HTTPS health differs")
    # Confirm the release pointer did not switch while files were being hashed.
    req(os.path.realpath(current) == release, "current-release changed during verification")
    for record, (docker, report) in zip(backends, observations):
        docker_after = json.loads(cmd("docker", "inspect", "--format", "{{json .}}", record["container_name"]))
        req(docker_after.get("Image") == docker.get("Image") and docker_after.get("State", {}).get("Pid") == report["pid"] and docker_after.get("RestartCount") == docker.get("RestartCount") and docker_after.get("State", {}).get("Health", {}).get("Status") == "healthy", "container changed during verification")
        if overlay:
            req(hfile("/proc/%d/exe" % report["pid"]) == record["binary_sha256"], "running binary changed during verification")
            check_backend_http(record)
            if record is overlay: check_overlay_standby(record, docker_after, report["pid"])
    roots_after, addresses_after = parse_nginx(cmd("nginx", "-T"), front["hostname"], overlay)
    req(roots_after == roots and addresses_after == addresses, "nginx routing changed during verification")
    result = {"runtime_live_verified": True, "backend": observations[0][1], "frontend": {"release_path": front["release_path"], "manifest_sha256": hfile(fmanifest), "assets": assets, "https_assets": https_assets, "https_index_asset": front["index_asset"], "nginx_root": front["nginx_root"], "active_upstreams": addresses}}
    if overlay: result["backends"] = [report for docker, report in observations]
    print(json.dumps(result, separators=(",", ":")))
try:
    main()
except E as e:
    print(json.dumps({"runtime_live_verified": False, "mismatch": str(e)})); sys.exit(1)
except Exception:
    print(json.dumps({"runtime_live_verified": False, "mismatch": "observer read or parse failed"})); sys.exit(1)
'''


if __name__ == "__main__":
    raise SystemExit("This module is invoked by release_sync.py; use --live-vps there.")
