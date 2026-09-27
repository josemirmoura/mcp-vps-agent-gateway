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

## Access profiles

The installer should offer three primary profiles.

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

## Installer UX

The installer should ask the operator:

1. Access profile: Project, Custom, or Whole host.
2. Allowed filesystem roots.
3. Filesystem capabilities.
4. Shell access and sandbox roots.
5. systemd unit scope/actions.
6. Docker/Compose scope/actions.
7. Network policy.
8. Administrative capabilities.
9. Approval/elevation behavior.
10. Authentication and public/private MCP exposure.

It then renders a policy file, displays a human-readable summary, asks for explicit confirmation, validates the policy, and only then activates it.

The policy remains editable later without reinstalling the binaries.

## Packaging

The desired product experience is one-command installation.

However, the runtime should preserve the privilege boundary:

~~~text
Gateway: containerized or native, non-root
Broker: native host service, privileged, local-only
~~~

Running the Broker inside a highly privileged container by default would require host filesystem/systemd/Docker access and would weaken the clean host privilege boundary.

Recommended production packaging:

- native Broker binary + systemd unit
- Gateway either native or containerized
- one installer/bootstrap command that installs both
- generated server-side policy
- optional Docker/Compose assets for environments where that is appropriate

A fully privileged all-Docker mode may exist for disposable labs, but should not be the production default.

The product goal is **one-command installation**, not "everything must execute inside one container".
