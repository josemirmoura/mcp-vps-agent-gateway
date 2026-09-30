# Architecture

## Objectif

Donner à un client IA un accès opérationnel utile à une VPS Linux tout en gardant la VPS, et non le modèle, comme autorité de sécurité.

> **Le LLM n’est jamais la frontière de sécurité.**

## Runtime canonique

Deux processus Go du projet :

~~~text
ChatGPT / client MCP
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| non privilégié              |
| MCP + auth + schemas        |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privilégié, local-only      |
| policy autoritative         |
| SQLite + locks + jobs       |
| secrets + audit             |
| files + systemd + Docker    |
| exécution sandboxed         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Reverse proxy ou tunnel privé supporté = infrastructure d’entrée, pas troisième service.

## Pourquoi Go

SDK Go MCP officiel Tier 1, compatible MCP 2026-07-28. Une langue réduit packaging, dépendances et maintenance tout en conservant la frontière de privilège par processus.

## Gateway

Non-root. Peut exposer MCP, valider OAuth/OIDC et schemas, normaliser tools/resources, faire des preflights non autoritatifs et appeler Broker via socket Unix.

Ne doit jamais être root, recevoir Docker socket, ouvrir SQLite privilégié, lire secrets plaintext, devenir autorité ou approuver sa propre élévation.

## Broker

Frontière privilégiée. Chaque appel est réautorisé contre :

~~~text
subject
+ tool canonique
+ resource canonique
+ action
+ policy courante
+ grant/lease/job si requis
~~~

Broker possède policy, SQLite, idempotency, locks, jobs, filesystem sûr, Docker/systemd, sandbox, secrets et audit. Gateway est un deputy non fiable.

## Modèle de capacités

Un catalogue large ; autorité effective par policy. Une racine, plusieurs ou Whole Host.

Plafond physique et racines logiques séparés : le plafond est une limite, pas un grant. Standard peut découvrir seulement les noms de dossiers immédiats. Les délégations dynamiques appartiennent au Broker, liées au subject, avec `read`/`work`/`compose` et TTL optionnel.

Créer une requête n’autorise rien. Avec MCP elicitation, le client rend la confirmation native. Broker valide request, subject et token one-time. Les tools de confirmation ne sont pas model-visible. Sans elicitation, fail-closed vers fallback opérateur. Revocation prend effet sans restart.

Les fichiers secrets sont une frontière imbriquée : `.env` demande un grant temporaire exact-path. Les templates restent normaux. Le sandbox shell masque les paths protégés.

Filesystem n’est qu’un axe ; systemd, Docker, shell, réseau et admin sont séparés. Voir [product-model.md](product-model.md).

## Policy / Approval

Policy Engine est un module du Broker. Policies deny-by-default. Presets Controlled, Scoped, Full ; Full off par défaut.

Routine Scoped sans interruption. L’élévation temporaire est confirmée hors canal d’action. Elicitation native préférée, Broker lie approval au subject + nonce one-time et empêche replay/autoapproval. Route admin séparée en fallback/MFA.

## Full

Expanse en capacités explicites :

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Réseau sans restriction séparé. Full temporaire, révocable et gated.

## Transport

MCP Streamable HTTP sur endpoint HTTPS stable. Core stateless : pas de WebSocket/session custom. Jobs/leases ont handles explicites ; SDK gère negotiation.

## Filesystem

Jamais de string-prefix authorization. Préférer `openat2` restrictif ; fallback directory-FD sécurisé avec `openat/fstatat/O_NOFOLLOW` ou fail closed. Jamais de repli silencieux vers strings.

## Isolation

systemd transient units, cgroups, NoNewPrivileges, PrivateTmp, restrictions FS, MemoryMax, TasksMax, deadline, output limit, cancellation. Landlock en défense supplémentaire.

## Docker

Gateway ne reçoit jamais Docker socket. Opérations Docker typées du Broker contre stacks/actions canoniques.

## Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Modèle durable interne, indépendant d’HTTP. MCP Tasks pourra être adapté ensuite.

## État

SQLite uniquement Broker, transactions courtes. Approvals, leases, jobs, idempotency, locks, audit metadata.

Locks avec resource, owner/action_id, fencing_token monotone, expires_at. Un ancien owner ne libère pas un token plus récent.

Journal d’idempotence :

~~~text
PENDING -> external effect -> DONE
~~~

Commit PENDING avant effet ; crash intermédiaire déclenche reconciliation, pas replay aveugle.

## Secrets

Préférer fichiers root-owned hors repo et systemd credentials. Broker résout les refs et injecte uniquement au processus cible. Gateway/tools n’exposent pas plaintext.

## Audit

Gate 1 : audit local structuré.
Scoped production : séquence/intégrité durables.
Avant Full : hash chain tamper-evident, checkpoint/forward distant, recovery testé.

## Downstream

Upstreams allowlisted out-of-band, noms namespaced déterministes, changements fingerprinted/reviewed, résultats untrusted. Les résultats ne changent jamais policy, leases, registration ou secrets.

## Non-objectifs initiaux

Pas gateway universel, control plane multi-tenant, scheduler distribué, root shell générique, Kubernetes, remplacement SSH ou support de chaque client.

> Premier objectif : prouver sûrement qu’un client IA réel peut exécuter une petite opération utile et auditée sur une VPS.
