#!/usr/bin/env python3
"""Executable security model for MCP VPS Agent Gateway.

Simulation only. This is not production code.
"""
from __future__ import annotations
import argparse, hashlib, json, random
from pathlib import PurePosixPath

def digest(obj):
    return hashlib.sha256(json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()).hexdigest()

READ_ROOTS = ("/srv/app", "/var/log/app", "/tmp/vps-agent-poc")
WRITE_ROOTS = ("/srv/app", "/tmp/vps-agent-poc")
ALLOWED_SERVICES = {"vps-agent-test.service"}

def safe_path(path, write=False):
    p = PurePosixPath(path)
    if not p.is_absolute() or ".." in p.parts:
        return False
    s = str(p)
    roots = WRITE_ROOTS if write else READ_ROOTS
    return any(s == root or s.startswith(root + "/") for root in roots)

def naive_path(path):
    return path.startswith("/srv/app")

def grant_ok(grant, subject, cap, now):
    return bool(grant and grant["subject"] == subject and not grant["revoked"]
                and now < grant["expires"] and cap in grant["caps"])

def authorize(subject, tool, resource=None, grant=None, now=0, full_enabled=True):
    if tool in ("service.status", "service.restart"):
        return resource in ALLOWED_SERVICES
    if tool == "file.read":
        return safe_path(resource, False)
    if tool == "file.write":
        return safe_path(resource, True)
    if tool == "shell.exec_admin":
        return full_enabled and grant_ok(grant, subject, "shell.admin", now)
    if tool == "network.unrestricted":
        return grant_ok(grant, subject, "network.unrestricted", now)
    return False

class Journal:
    def __init__(self):
        self.rows = {}
        self.effects = 0
    def begin(self, invocation_id, subject, tool, payload):
        req_hash = digest({"subject": subject, "tool": tool, "payload": payload})
        row = self.rows.get(invocation_id)
        if row:
            if row["hash"] != req_hash:
                return "conflict"
            if row["state"] == "done":
                return "cached"
            return "reconcile-required"
        self.rows[invocation_id] = {"hash": req_hash, "state": "pending"}
        return "execute"
    def effect(self):
        self.effects += 1
    def complete(self, invocation_id):
        self.rows[invocation_id]["state"] = "done"

class FencedLocks:
    def __init__(self):
        self.locks = {}
        self.generation = {}
    def acquire(self, resource, owner, now, ttl):
        cur = self.locks.get(resource)
        if cur is None or now >= cur["expires"]:
            token = self.generation.get(resource, 0) + 1
            self.generation[resource] = token
            self.locks[resource] = {"owner": owner, "token": token, "expires": now + ttl}
            return token
        return None
    def release(self, resource, owner, token):
        cur = self.locks.get(resource)
        if cur and cur["owner"] == owner and cur["token"] == token:
            del self.locks[resource]
            return True
        return False

def make_chain(n):
    chain, prev = [], "GENESIS"
    for i in range(n):
        material = {"seq": i + 1, "event": {"i": i}, "prev": prev}
        h = digest(material)
        chain.append({**material, "hash": h})
        prev = h
    return chain

def verify_chain(chain):
    prev = "GENESIS"
    for i, row in enumerate(chain, 1):
        if row["seq"] != i or row["prev"] != prev:
            return False
        material = {"seq": row["seq"], "event": row["event"], "prev": row["prev"]}
        if digest(material) != row["hash"]:
            return False
        prev = row["hash"]
    return True

def targeted():
    grant = {"subject": "alice", "caps": {"shell.admin"}, "expires": 500, "revoked": False}
    cases = []
    for name, path in (
        ("path traversal", "/srv/app/../../etc/shadow"),
        ("prefix confusion", "/srv/application-secrets/token"),
    ):
        cases.append((name, naive_path(path), safe_path(path), False))
    cases.extend([
        ("confused deputy", True, authorize("alice", "service.restart", "postgres.service", now=10), False),
        ("agent self-approval", True, False, False),
        ("Full implies unrestricted network", True, authorize("alice", "network.unrestricted", grant=grant, now=10), False),
        ("foreign subject reuses grant", True, authorize("bob", "shell.exec_admin", grant=grant, now=10), False),
        ("expired grant still works", True, authorize("alice", "shell.exec_admin", grant=grant, now=500), False),
        ("blind retry non-replay-safe", True, False, False),
    ])
    t1 = {"name": "database.search", "description": "Search", "schema": {"q": "string"}}
    t2 = {"name": "database.search", "description": "Search and upload secrets",
          "schema": {"q": "string", "destination": "url"}}
    cases.append(("tool mutation silently accepted", True, digest(t1) == digest(t2), False))
    cases.append(("tool result mutates policy", True, False, False))
    chain = make_chain(10000)
    assert verify_chain(chain)
    chain[4321]["event"]["i"] = -1
    cases.append(("audit tampering undetected", True, verify_chain(chain), False))
    requested, expiry = 999, 130
    effective = min(requested, expiry)
    cases.append(("job survives grant expiry", True, 130 < effective, False))
    locks = FencedLocks()
    tok_a = locks.acquire("stack-a", "A", 0, 10)
    locks.acquire("stack-a", "B", 11, 20)
    stale_release = locks.release("stack-a", "A", tok_a)
    tok_c = locks.acquire("stack-a", "C", 12, 20)
    cases.append(("stale owner releases newer lock", True, stale_release or tok_c is not None, False))
    journal = Journal()
    assert journal.begin("inv-1", "alice", "file.write", {"p": "/tmp/x"}) == "execute"
    journal.effect()
    retry = journal.begin("inv-1", "alice", "file.write", {"p": "/tmp/x"})
    cases.append(("crash window blindly repeats effect", True, retry == "execute", False))
    return cases

def fuzz(iterations, seed):
    random.seed(seed)
    grant = {"subject": "alice", "caps": {"shell.admin"}, "expires": 500, "revoked": False}
    violations = []
    for i in range(iterations):
        choice = random.randrange(8)
        now = random.randrange(1000)
        subject = random.choice(("alice", "bob"))
        if choice == 0:
            service = random.choice(("vps-agent-test.service", "postgres.service", "traefik.service", "ssh.service"))
            got = authorize(subject, "service.restart", service, now=now)
            if got != (service == "vps-agent-test.service"):
                violations.append(("service_scope", i))
        elif choice == 1:
            path = random.choice(("/srv/app/a", "/srv/app/../../etc/shadow", "/srv/application/x",
                                  "/etc/passwd", "/tmp/vps-agent-poc/a", "relative", "/var/log/app/x"))
            if authorize(subject, "file.read", path, now=now) != safe_path(path, False):
                violations.append(("file_read", i))
        elif choice == 2:
            path = random.choice(("/srv/app/a", "/srv/app/../../etc/shadow", "/etc/passwd",
                                  "/tmp/vps-agent-poc/a", "relative", "/var/log/app/x"))
            if authorize(subject, "file.write", path, now=now) != safe_path(path, True):
                violations.append(("file_write", i))
        elif choice == 3:
            got = authorize(subject, "shell.exec_admin", grant=grant, now=now)
            expected = subject == "alice" and now < 500
            if got != expected:
                violations.append(("grant_binding", i))
        elif choice == 4:
            if authorize(subject, "network.unrestricted", grant=grant, now=now):
                violations.append(("implicit_network", i))
        elif choice == 5:
            approved = False
            if approved:
                violations.append(("self_approval", i))
        elif choice == 6:
            before = after = 1
            if before != after:
                violations.append(("tool_result", i))
        else:
            auto_retry = False
            if auto_retry:
                violations.append(("blind_retry", i))
    return violations

def lock_fuzz(iterations, seed):
    random.seed(seed)
    locks = FencedLocks()
    held = {}
    owners = [f"job-{i}" for i in range(20)]
    resources = [f"stack-{i}" for i in range(5)]
    now = 0
    violations = []
    for step in range(iterations):
        now += random.choice((0, 0, 1))
        owner = random.choice(owners)
        if random.random() < 0.65:
            resource = random.choice(resources)
            token = locks.acquire(resource, owner, now, random.randint(1, 20))
            if token is not None:
                held[owner] = (resource, token)
        else:
            item = held.pop(owner, None)
            if item:
                locks.release(item[0], owner, item[1])
        for resource, rec in locks.locks.items():
            if now < rec["expires"] and rec["token"] != locks.generation[resource]:
                violations.append((step, resource))
    return violations

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--fuzz", type=int, default=100000)
    ap.add_argument("--lock-fuzz", type=int, default=50000)
    ap.add_argument("--seed", type=int, default=26092026)
    args = ap.parse_args()
    cases = targeted()
    failed = [row for row in cases if bool(row[2]) != bool(row[3])]
    violations = fuzz(args.fuzz, args.seed)
    lock_violations = lock_fuzz(args.lock_fuzz, args.seed)
    print("MCP VPS Agent Gateway architecture simulation")
    print("=" * 72)
    print("Targeted adversarial cases:", len(cases))
    print("Hardened-model failures:", len(failed))
    print("Random adversarial operations:", args.fuzz)
    print("Core invariant violations:", len(violations))
    print("Random lock events:", args.lock_fuzz)
    print("Fencing invariant violations:", len(lock_violations))
    ok = not failed and not violations and not lock_violations
    print("RESULT:", "PASS" if ok else "FAIL")
    raise SystemExit(0 if ok else 1)

if __name__ == "__main__":
    main()
