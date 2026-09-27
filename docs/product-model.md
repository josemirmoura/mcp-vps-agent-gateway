# Product model: full toolbox, scoped authority

## Core product decision

Ship one complete capability set and control authority through server-side policy.

The installed software should not have separate "limited", "project", or "full" binaries.

Instead:

~~~text
same binaries
+ same MCP tool catalog
+ different policy
= different effective authority
~~~

The LLM never decides the scope.

## User-owned scope

The operator decides exactly how much of the VPS is delegated to MCP.

The package may provide policy examples for convenience, but examples never define authority by themselves.

The authoritative configuration is always the explicit policy selected and confirmed by the user.

Examples:

~~~text
one directory:
/opt/my-app

several directories:
/opt/app-a
/var/www/site
/srv/data

whole filesystem:
/
~~~

The same principle applies independently to systemd units, Docker resources, network destinations, package management, users/groups and other administrative resources.

## Convenience profiles

The installer may offer three shortcuts:

### Project

Autonomous access only inside one project root, for example:

~~~text
/opt/my-app
~~~

Typical enabled capabilities:

- filesystem CRUD inside the project root
- scoped shell execution with cwd inside the project root
- selected Docker stack/container operations
- selected systemd units
- selected outbound network destinations

### Custom

The operator selects multiple filesystem roots and resource groups, for example:

~~~text
/opt/app-a
/var/www/site
/srv/data
~~~

The same capability catalog is available, but policy limits where each capability can act.

### Whole host

The operator explicitly authorizes host-wide resources.

This is a policy choice, not a different build.

Whole-host mode can enable capabilities such as:

- filesystem access to /
- broad systemd administration
- Docker administration
- package management
- user/group management
- firewall/network administration
- temporary administrative shell

Dangerous capabilities still remain explicit and auditable.

## Capability catalog target

### Filesystem

- file.list
- file.stat
- file.read
- file.mkdir
- file.write
- file.patch
- file.copy
- file.move
- file.remove
- file.remove_recursive
- file.hash
- file.chmod
- file.chown

Capabilities exist in the product even when policy disables them.

Recursive deletion and broad permission changes should be separate capabilities so policy can distinguish ordinary file work from destructive administration.

### Commands and jobs

- shell.exec
- job.start
- job.status
- job.tail
- job.cancel

shell.exec must execute inside a server-enforced sandbox. A filesystem scope written only in YAML is insufficient by itself for shell commands.

The sandbox must translate policy into OS restrictions such as:

- allowed cwd roots
- ProtectSystem
- ReadWritePaths / ReadOnlyPaths
- ProtectHome
- PrivateTmp
- cgroups
- MemoryMax
- TasksMax
- timeout
- output limits
- network policy

### systemd

- service.list
- service.status
- service.logs
- service.start
- service.stop
- service.restart
- service.reload
- service.enable
- service.disable

Policy limits canonical unit names/patterns and actions.

### Docker / Compose

- docker.list
- docker.inspect
- docker.logs
- docker.start
- docker.stop
- docker.restart
- compose.config
- compose.pull
- compose.up
- compose.down

Prefer typed operations. Do not expose the Docker socket to the Gateway.

### Diagnostics

- system.info
- system.health
- system.disk
- system.memory
- process.list
- process.inspect
- network.listen
- network.check
- journal.read

### Administration

Available in the product but normally disabled unless explicitly selected:

- package.update
- package.install
- package.remove
- user/group administration
- firewall administration
- file.chmod / file.chown beyond project-safe ranges
- shell.exec_admin

## Scope is multidimensional

Filesystem path is only one axis.

A policy can separately scope:

~~~text
filesystem roots
systemd units
Docker resources/stacks
shell cwd roots
network destinations
package-manager actions
users/groups
firewall changes
temporary admin capabilities
~~~

A project profile may have full CRUD inside /opt/my-app while having no authority over nginx, Docker, apt or outbound Internet.

## chmod and destructive operations

The implementation may expose file.chmod, including mode 0777, when policy allows it.

Default profiles should not silently enable world-writable permissions.

Similarly:

- file.remove handles ordinary deletion
- file.remove_recursive is a distinct destructive capability
- host-wide chmod/chown is a distinct administrative decision

The product supports the operation; policy decides whether the current installation may use it.

## Configuration UX

The operator or assisting AI edits the declarative policy to choose:

1. Which exact filesystem roots/resources are delegated to MCP.
2. Optional convenience preset: Project, Custom, or Whole host.
3. Allowed filesystem roots.
4. Filesystem capabilities.
5. Shell access and sandbox roots.
6. systemd unit scope/actions.
7. Docker/Compose scope/actions.
8. Network policy.
9. Administrative capabilities.
10. Approval/elevation behavior.
11. Authentication and public/private MCP exposure.

`config/policy.yaml` is the authoritative, human/AI-readable configuration. `docker compose config -q` and runtime validation reject invalid package configuration before use.

The policy remains editable later without reinstalling the binaries.

## Packaging

The official product packaging is Docker Compose.

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

The Broker container is **not** a security sandbox around host administration. It is the privileged server-side security boundary packaged in Docker.

Therefore:

- Docker provides reproducible packaging and lifecycle.
- the Gateway remains separated and unprivileged;
- the Broker receives the host root only because it must implement the operator-authorized host actions;
- the Broker exposes no remote TCP control API;
- server-side policy decides what part/resources of the VPS the MCP may use;
- mounting /host does not grant the LLM authority by itself;
- the same package supports one project, several resources, or the whole host through policy.

The primary operator flow is intentionally terminal- and AI-friendly:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/vps-agent-sandbox
bash scripts/init.sh --scope /opt/vps-agent-sandbox
# optionally refine config/policy.yaml and .env
docker compose up -d --build
bash scripts/verify.sh
bash scripts/setup-integrated-auth.sh --domain mcp.example.com
bash scripts/connect-chatgpt.sh
~~~

There is no separate interactive installer or alternate native-install product flow.

## Installation is not complete until ChatGPT connectivity is verified

The ChatGPT Web tutorial is an earlier installation step. It teaches the operator how to connect, but it does **not** complete the installation.

Installation completes only after the connection has been established and verified end-to-end.

After runtime installation and policy activation, it must present:

~~~text
MCP endpoint:
https://<host>/mcp

Authentication:
<configured method>

Effective scope:
<human-readable policy summary>

Next:
Connect this MCP to ChatGPT Web
~~~

The first-run sequence must:

1. verify the MCP HTTPS endpoint is reachable;
2. verify authentication;
3. run a harmless server-side discovery/read test;
4. show the exact effective authority selected by the user;
5. provide the current ChatGPT Web connection tutorial for the supported integration route;
6. wait for the operator to connect ChatGPT Web;
7. require a harmless tool call from ChatGPT itself;
8. verify that the call reached the expected subject, passed policy, executed successfully, and appears in Broker audit;
9. only then mark installation complete.

Because ChatGPT product surfaces can change independently from this project, the tutorial must be versioned and revalidated against current official OpenAI documentation at release/install time.

The desired final experience is:

~~~text
clone / release bundle
 -> choose exactly what MCP may control
 -> start Gateway + Broker with Docker Compose
 -> validate policy and security
 -> bootstrap integrated OAuth + HTTPS endpoint
 -> show ChatGPT Web connection tutorial
 -> user connects ChatGPT
 -> verify real end-to-end ChatGPT tool call
 -> verify Broker audit
 -> installation complete
~~~
