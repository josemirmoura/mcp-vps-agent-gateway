# Product model: full toolbox, scoped authority

## Core decision

One complete capability set を配布し、authority は server-side policy で制御します。

Separate limited/project/full binaries はありません。

~~~text
same binaries
+ same MCP tool catalog
+ different policy
= different effective authority
~~~

LLM は scope を決めません。

## User-owned scope

Operator が VPS のどこまでを delegate するか決めます。Example policy は権限を生まず、explicit confirmed policy だけが authoritative。

~~~text
one directory:
/opt/my-app

several:
/opt/app-a
/var/www/site
/srv/data

whole filesystem:
/
~~~

同じ原則を systemd、Docker、network、packages、users/groups、admin resources に独立適用。

## Profiles

### Standard

Multi-project host 推奨。Physical ceiling default `/opt`、installation 時は project root を authorize しません。

Portico は直下の directory name だけ discover。内容は `read`/`work`/`compose` approval まで locked。Protected secret はさらに nested boundary。

### Project

`/opt/my-app` など1 root 内だけ autonomous: filesystem CRUD、Scoped shell、selected Docker/systemd、selected network destinations。

### Advanced/static

Advanced operator は複数 static roots/resource groups を policy で定義可能。Guided Standard は initial YAML editing より runtime delegation を優先。

### Whole host

Host-wide resources を explicit authorize。別 build ではありません。`/`、broad systemd、Docker、packages、users/groups、firewall/network、temporary admin shell など。ただし dangerous capability は explicit/auditable。

## Capability catalog

### Filesystem

`file.list/stat/read/mkdir/write/patch/copy/move/remove/remove_recursive/hash/chmod/chown`.

Policy disabled でも capability は product に存在。Recursive delete と broad permission change は separate capabilities。

### Commands/jobs

`shell.exec`, `job.start/status/tail/cancel`.

Shell は server-enforced sandbox。YAML の path scope だけでは不十分。Policy を cwd roots、ProtectSystem、ReadWritePaths/ReadOnlyPaths、ProtectHome、PrivateTmp、cgroups、MemoryMax、TasksMax、timeout、output limits、network policy に変換。

### systemd

`service.list/status/logs/start/stop/restart/reload/enable/disable`。Canonical unit/action を policy で制限。

### Docker/Compose

`docker.list/inspect/logs/start/stop/restart`, `compose.config/pull/up/down`。Typed operation を優先し Gateway に Docker socket を渡しません。

### Diagnostics

`system.info/health/disk/memory`, `process.list/inspect`, `network.listen/check`, `journal.read`.

### Administration

Product にはあるが explicit selection まで disabled: package management、users/groups、firewall、broad chmod/chown、`shell.exec_admin`.

## Multidimensional scope

Filesystem は1軸。Roots、systemd units、Docker stacks、shell cwd、network destinations、package actions、users/groups、firewall、temporary admin を別々に scope。

Project が `/opt/my-app` full CRUD でも nginx/Docker/apt/Internet は zero authority にできます。

## chmod / destructive

Policy が許せば `file.chmod` 0777 も可能ですが defaults は world-writable を silent enable しません。`file.remove_recursive` は separate destructive capability、host-wide chmod/chown は separate admin decision。

## Configuration UX

`config/policy.yaml` が authoritative human/AI-readable config。Roots/resources、preset、filesystem、shell/sandbox、systemd、Docker/Compose、network、admin、approval/elevation、auth/exposure を決定。Compose/runtime が invalid config を reject。Reinstall なしで変更可能。

## Packaging

~~~text
Gateway container
  non-root
  no /host
  no Docker socket
        |
        | Unix socket
        v
Broker container
  privileged host-control boundary
  host mounted at /host
        |
        v
VPS
~~~

Broker container は security sandbox ではなく privileged server-side boundary。Docker は packaging/lifecycle、Gateway は unprivileged、Broker は authorized host action のためだけ host root を持ち remote TCP control API はなし。Policy が authority を決め、`/host` mount だけでは LLM に権限を与えません。

Primary flow:

~~~bash
bash scripts/install.sh
~~~

Bootstrap、Compose、verification、OAuth、ChatGPT connection の transparent orchestration。Standard/Project/Whole Host と effective authority を表示し resume 可能。Individual commands も利用可能。[installation-contract.md](installation-contract.md) が supported boundary。

## ChatGPT verification まで incomplete

Tutorial 表示だけでは complete ではありません。Runtime/policy 後に endpoint/auth/effective scope を表示。

First-run:

1. HTTPS;
2. auth;
3. harmless server-side test;
4. effective authority;
5. current ChatGPT tutorial;
6. wait for connection;
7. require harmless call from ChatGPT;
8. verify subject/policy/execution/audit;
9. only then complete.

ChatGPT surface は独立して変化するため tutorial は versioned/revalidated。

~~~text
clone / bundle
 -> choose authority
 -> start Gateway + Broker
 -> validate policy/security
 -> OAuth + HTTPS
 -> ChatGPT tutorial
 -> connect ChatGPT
 -> verify real E2E call
 -> verify audit
 -> installation complete
~~~
