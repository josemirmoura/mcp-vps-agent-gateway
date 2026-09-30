# Model produk: toolbox penuh, authority Scoped

## Keputusan inti

Kirim satu capability set lengkap dan kontrol authority melalui server-side policy.

Tidak ada binary limited/project/full terpisah:

~~~text
same binaries
+ same MCP tool catalog
+ different policy
= different effective authority
~~~

LLM tidak pernah menentukan scope.

## User-owned scope

Operator memutuskan persis berapa banyak VPS yang didelegasikan. Contoh policy tidak memberi authority; explicit confirmed policy yang authoritative.

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

Prinsip sama berlaku terpisah untuk systemd, Docker, network, package, users/groups, dan admin resources.

## Profiles

### Standard

Rekomendasi untuk multi-project host. Physical ceiling default `/opt`, tanpa project root authorized saat instalasi.

Portico hanya discover immediate directory names. Isi terkunci sampai operator approve root dengan `read`, `work`, atau `compose`. Protected secrets tetap nested boundary.

### Project

Autonomous access hanya dalam root seperti `/opt/my-app`: filesystem CRUD, Scoped shell dengan cwd di root, selected Docker/systemd, selected network destinations.

### Advanced/static policy

Operator advanced dapat mendefinisikan banyak static roots/resource groups. Guided Standard lebih memilih runtime delegation daripada memaksa edit YAML awal.

### Whole host

Operator secara eksplisit mengotorisasi host-wide resources. Ini policy, bukan build berbeda. Dapat mencakup `/`, broad systemd, Docker, package management, users/groups, firewall/network, temporary admin shell. Dangerous capabilities tetap explicit/auditable.

## Capability catalog

### Filesystem

`file.list/stat/read/mkdir/write/patch/copy/move/remove/remove_recursive/hash/chmod/chown`.

Capability tetap ada walau policy disable. Recursive delete dan broad permission change dipisah.

### Commands/jobs

`shell.exec`, `job.start/status/tail/cancel`.

Shell wajib server-enforced sandbox. Scope YAML saja tidak cukup. Policy diterjemahkan ke cwd roots, ProtectSystem, ReadWritePaths/ReadOnlyPaths, ProtectHome, PrivateTmp, cgroups, MemoryMax, TasksMax, timeout, output limits, network policy.

### systemd

`service.list/status/logs/start/stop/restart/reload/enable/disable`; policy membatasi canonical units/actions.

### Docker/Compose

`docker.list/inspect/logs/start/stop/restart`, `compose.config/pull/up/down`. Prefer typed operations; Docker socket tidak pernah ke Gateway.

### Diagnostics

`system.info/health/disk/memory`, `process.list/inspect`, `network.listen/check`, `journal.read`.

### Administration

Tersedia tetapi default off: package management, users/groups, firewall, broad chmod/chown, `shell.exec_admin`.

## Scope multidimensional

Filesystem hanya satu axis. Roots, systemd units, Docker stacks, shell cwd, network destinations, package actions, users/groups, firewall, temporary admin capability di-scope terpisah.

Project dapat full CRUD di `/opt/my-app` tetapi zero authority atas nginx, Docker, apt, Internet.

## chmod / destructive ops

`file.chmod`, termasuk 0777, dapat tersedia jika policy mengizinkan. Default tidak silent-enable world-writable. `file.remove_recursive` capability destructive terpisah; host-wide chmod/chown keputusan admin terpisah.

## Configuration UX

`config/policy.yaml` authoritative human/AI-readable config: resources, preset, filesystem, shell/sandbox, systemd, Docker/Compose, network, administration, approval/elevation, auth/exposure. Compose/runtime reject invalid config dan policy dapat diedit tanpa reinstall.

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

Broker bukan security sandbox, melainkan privileged server-side boundary yang dipaketkan Docker. Docker menyediakan packaging/lifecycle; Gateway tetap unprivileged; Broker menerima host root hanya untuk tindakan authorized, tidak punya remote TCP control API, policy menentukan resources. Mount `/host` tidak otomatis memberi authority ke LLM.

Primary flow:

~~~bash
bash scripts/install.sh
~~~

Transparent orchestration atas bootstrap, Compose, verification, OAuth, ChatGPT connection. Standard/Project/Whole Host, effective authority, resumable. Individual commands tetap tersedia. [installation-contract.md](installation-contract.md) menetapkan supported boundary.

## Instalasi selesai hanya setelah ChatGPT verified

Tutorial saja belum cukup. Setelah runtime/policy, tampilkan endpoint, auth, effective scope.

First-run:

1. verify HTTPS;
2. verify auth;
3. harmless server-side test;
4. show authority;
5. current ChatGPT tutorial;
6. wait for connection;
7. require harmless call from ChatGPT;
8. verify subject/policy/execution/audit;
9. only then complete.

ChatGPT surface berubah independen, jadi tutorial versioned/revalidated.

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
