"""Exercise the shipped OIDC executable with synthetic local identity services."""
import http.server
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

IDENTITY = "8c0b7f5a-4dce-4d13-b6ad-6b2df2c1d10c"
ACTOR = "1bcf5bd1-5f3e-4f01-9125-0edc93e5f5b1"


def fixture():
    captured = {"login": [], "consent": [], "whoami": 0, "api": 0, "requests": []}

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            path = urllib.parse.urlparse(self.path).path
            if path == "/assertions":
                return self.reply(captured)
            captured["requests"].append({"method": "GET", "path": path})
            if path in ("/health/ready", "/admin/health/ready", "/health/alive", "/admin/health/alive", "/version"):
                return self.reply({"status": "ok", "version": "synthetic"})
            if path == "/sessions/whoami":
                captured["whoami"] += 1
                if self.headers.get("Cookie") != "ory_kratos_session=synthetic-session":
                    return self.reply({"error": "invalid session"}, 401)
                return self.reply({"identity": {"id": IDENTITY}})
            if path == "/admin/identities/" + IDENTITY:
                return self.reply({"id": IDENTITY, "schema_id": "default", "schema_url": "http://fixture:8081/schema",
                                   "traits": {"email": "oidc@example.invalid", "display_name": "Synthetic User"},
                                   "metadata_public": {"email_verified": True, "accepted_terms": True}})
            if path == "/admin/oauth2/auth/requests/login":
                challenge = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query).get("login_challenge", [""])[0]
                if challenge == "reject":
                    return self.reply({"error": "invalid_request", "error_description": "synthetic rejection"}, 404)
                return self.reply({"challenge": challenge, "request_url": "http://fixture:8081/authorize", "skip": False, "subject": "", "client": {"client_id": "synthetic"}})
            if path == "/admin/oauth2/auth/requests/consent":
                return self.reply({"challenge": "consent", "subject": IDENTITY, "context": {"identity_id": IDENTITY},
                                   "requested_scope": ["openid", "profile", "email", "alkemio"], "client": {"client_id": "synthetic"}})
            return self.reply({"error": "unexpected fixture path", "path": path}, 500)

        def do_PUT(self):
            captured["requests"].append({"method": "PUT", "path": self.path})
            payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
            path = urllib.parse.urlparse(self.path).path
            if path == "/admin/oauth2/auth/requests/login/accept":
                captured["login"].append(payload)
            elif path == "/admin/oauth2/auth/requests/consent/accept":
                captured["consent"].append(payload)
            else:
                return self.reply({"error": "unexpected write"}, 500)
            return self.reply({"redirect_to": "http://fixture:8081/callback"})

        def do_POST(self):
            captured["requests"].append({"method": "POST", "path": self.path})
            captured["api"] += 1
            self.reply({"error": "database hit must not call API fallback"}, 500)

        def reply(self, body, status=200):
            raw = json.dumps(body).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(raw)

    http.server.ThreadingHTTPServer(("0.0.0.0", 8081), Handler).serve_forever()


def smoke(image):
    evidence = Path(os.environ["EVIDENCE"]).resolve()
    evidence.mkdir(parents=True, exist_ok=True)
    if not image.startswith("sha256:"):
        raise ValueError("IMAGE must be an immutable local image ID")
    fixture_image, postgres_image = os.environ["FIXTURE_IMAGE"], os.environ["POSTGRES_IMAGE"]
    for ref in (image, fixture_image, postgres_image):
        if not ref.startswith("sha256:"):
            raise ValueError("fixture images must be immutable local image IDs")
    prefix = "forge091-oidc-" + uuid.uuid4().hex[:12]
    names = {kind: prefix + "-" + kind for kind in ("network", "volume", "fixture", "database", "service", "maintenance")}
    created = {"containers": [], "networks": [], "volumes": []}
    result = {"image": image, "pass": False, "checks": [], "cleanup": []}
    urllib.request.install_opener(urllib.request.build_opener(urllib.request.ProxyHandler({})))

    def docker(*args, input=None):
        # Parsed output (container IDs, JSON) comes from stdout alone; stderr is kept as diagnostics.
        process = subprocess.run(["docker", *args], input=input, text=True, capture_output=True)
        if process.stderr:
            with (evidence / "docker-stderr.log").open("a") as log:
                log.write(json.dumps(args) + "\n" + process.stderr + "\n")
        if process.returncode:
            raise subprocess.CalledProcessError(process.returncode, process.args, process.stdout, process.stderr)
        return process.stdout.strip()

    def request(base, path, cookie=None):
        req = urllib.request.Request(base + path, headers={"Cookie": cookie} if cookie else {})
        try:
            response = urllib.request.urlopen(req, timeout=3)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            return response.status, dict(response.headers), response.read().decode()

    def wait_ready(check):
        for _ in range(80):
            try:
                if check():
                    return
            except (OSError, subprocess.CalledProcessError):
                pass
            time.sleep(0.25)
        raise RuntimeError("fresh fixture did not become ready")

    def run_container(kind, *args):
        cid = docker("create", "--name", names[kind], "--network", names["network"], *args)
        created["containers"].append(cid)
        (evidence / "created.json").write_text(json.dumps(created, indent=2))
        docker("start", cid)
        return cid

    def port(cid, container_port):
        address = json.loads(docker("inspect", cid))[0]["NetworkSettings"]["Networks"][names["network"]]["IPAddress"]
        return "http://" + address + ":" + container_port.split("/")[0]

    if docker("context", "inspect", "--format", "{{.Endpoints.docker.Host}}") != "unix:///var/run/docker.sock":
        raise RuntimeError("only the default local Unix Docker endpoint is allowed")
    for resource in ("container", "network", "volume"):
        rows = docker(resource, "ls", *( ["-a"] if resource == "container" else []), "--format", "{{json .}}")
        (evidence / (resource + "-before.jsonl")).write_text(rows)
        if any(value in rows for value in names.values()):
            raise RuntimeError("resource name collision")
    staged_fixture = evidence / "fixture.py"
    staged_fixture.write_text(Path(__file__).read_text())
    planned = {"names": names, "images": [image, fixture_image, postgres_image],
               "ports": "No published host ports; host connects directly to fresh internal bridge addresses", "storage": "fresh named PostgreSQL volume",
               "mounts": [str(staged_fixture)], "created": created}
    (evidence / "resources.json").write_text(json.dumps(planned, indent=2))
    try:
        created["networks"].append(docker("network", "create", "--internal", names["network"]))
        docker("volume", "create", names["volume"])
        created["volumes"].append(names["volume"])
        fixture_id = run_container("fixture", "--network-alias", "fixture",
                                   "-v", str(staged_fixture) + ":/fixture.py:ro", fixture_image,
                                   "python3", "-I", "/fixture.py", "--fixture")
        db = run_container("database", "--network-alias", "database", "-e", "POSTGRES_PASSWORD=synthetic-only",
                           "-e", "POSTGRES_USER=oidc", "-e", "POSTGRES_DB=oidc", "-v", names["volume"] + ":/var/lib/postgresql/data", postgres_image)
        wait_ready(lambda: docker("exec", db, "pg_isready", "-h", "127.0.0.1", "-U", "oidc", "-d", "oidc").endswith("accepting connections"))
        sql = 'CREATE TABLE "user" (id uuid PRIMARY KEY, "authenticationID" uuid UNIQUE NOT NULL);\n'
        sql += f'INSERT INTO "user" VALUES (\'{ACTOR}\', \'{IDENTITY}\');\n'
        docker("exec", "-i", db, "psql", "-v", "ON_ERROR_STOP=1", "-U", "oidc", "-d", "oidc", input=sql)
        fixture_url = port(fixture_id, "8081/tcp")
        wait_ready(lambda: request(fixture_url, "/assertions")[0] == 200)
        env = {"HYDRA_ADMIN_URL": "http://fixture:8081", "KRATOS_ADMIN_URL": "http://fixture:8081",
               "KRATOS_PUBLIC_URL": "http://fixture:8081", "ALKEMIO_SERVER_URL": "http://fixture:8081",
               "ALLOW_INSECURE_HTTP": "true", "DATABASE_HOST": "database", "DATABASE_USERNAME": "oidc",
               "DATABASE_PASSWORD": "synthetic-only", "DATABASE_NAME": "oidc"}

        def service(kind, maintenance=False):
            args = []
            for key, value in env.items():
                args.extend(["-e", "OIDC_" + key + "=" + value])
            if maintenance:
                args.extend(["-e", "OIDC_MAINTENANCE_MODE=true", "-e", "OIDC_MAINTENANCE_RETRY=30"])
            cid = run_container(kind, *args, image)
            base = port(cid, "8080/tcp")
            wait_ready(lambda: request(base, "/health/live")[0] == 200)
            return base

        base = service("service")
        # Do not follow Hydra redirects: inspect the executable's response itself.
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                return None
        urllib.request.install_opener(urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect))
        assert request(base, "/health/ready")[0] == 200
        status, headers, _ = request(base, "/v1/oidc/login?login_challenge=login", "ory_kratos_session=synthetic-session")
        assert status == 302 and headers.get("Location") == "http://fixture:8081/callback", (status, headers)
        status, headers, _ = request(base, "/v1/oidc/consent?consent_challenge=consent")
        assert status == 302 and headers.get("Location") == "http://fixture:8081/callback", (status, headers)
        state = json.loads(request(fixture_url, "/assertions")[2])
        assert state["login"][0]["subject"] == IDENTITY and state["whoami"] == 1, state
        for token in ("id_token", "access_token"):
            assert state["consent"][0]["session"][token]["alkemio_actor_id"] == ACTOR, state
        assert state["api"] == 0, state
        result["checks"].extend(["Kratos session and Hydra login acceptance", "consent actor claims from real PostgreSQL lookup without API fallback"])
        assert request(base, "/v1/oidc/login")[0] == 400
        assert request(base, "/v1/oidc/login?login_challenge=reject")[0] == 404
        result["checks"].append("missing and rejected login challenges fail closed")
        maint = service("maintenance", True)
        before = request(fixture_url, "/assertions")[2]
        status, headers, _ = request(maint, "/v1/oidc/login?login_challenge=maintenance")
        assert status == 503 and headers.get("Retry-After") == "30", (status, headers)
        assert request(fixture_url, "/assertions")[2] == before
        result["checks"].append("maintenance rejects before upstream calls with Retry-After")
        result["fixture_assertions"] = state
        result["pass"] = True
    finally:
        cleanup_errors = []
        for cid in reversed(created["containers"]):
            try:
                (evidence / (cid + ".inspect.json")).write_text(docker("inspect", cid))
                (evidence / (cid + ".log")).write_text(docker("logs", cid))
            except subprocess.CalledProcessError as error:
                cleanup_errors.append(str(error))
            try:
                docker("rm", "-f", cid)
                result["cleanup"].append({"container": cid, "removed": True})
            except subprocess.CalledProcessError as error:
                cleanup_errors.append(str(error))
        for kind, command in (("volumes", "volume"), ("networks", "network")):
            for resource in created[kind]:
                try:
                    docker(command, "rm", resource)
                    result["cleanup"].append({command: resource, "removed": True})
                except subprocess.CalledProcessError as error:
                    cleanup_errors.append(str(error))
        result["cleanup_errors"] = cleanup_errors
        if cleanup_errors:
            result["pass"] = False
        (evidence / "created.json").write_text(json.dumps(created, indent=2))
        (evidence / "smoke-result.json").write_text(json.dumps(result, indent=2))
        if cleanup_errors:
            raise RuntimeError("fixture cleanup or evidence capture failed: " + str(cleanup_errors))



if __name__ == "__main__":
    if len(sys.argv) == 2 and sys.argv[1] == "--fixture":
        fixture()
    else:
        smoke(sys.argv[1])
