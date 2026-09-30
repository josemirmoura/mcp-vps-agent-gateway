# Politique de release et de version

Portico MCP suit Semantic Versioning.

## Channels

### Stable

Les releases stables utilisent des tags comme `v0.1.0`, `v0.1.1` et `v0.2.0`. Elles sont le canal normal; `main` n’est pas production.

### Release candidate

Les RC utilisent des tags SemVer de pré-release comme `v0.1.0-rc.1` pour l’acceptation finale.

### Développement

`main` est la branche de développement et n’est pas la cible par défaut de `scripts/update.sh`.

## RC5

Le candidat actuel est `0.1.0-rc.5`. RC5 ajoute discovery du plafond sans accès au contenu, protection des secrets, elicitation MCP native, annotations de sécurité et promotion manuelle GitHub. Les tags RC antérieurs restent immuables.

Stable `v0.1.0` n’est créé qu’après le gate final d’acceptation sur installation propre.

## Publication

Chemin préféré: **GitHub Actions → release → Run workflow** sur `main`. Le workflow valide source, tests et contrat avant de créer le tag, publier les images multi-architecture et la GitHub Release.

## Artefacts

Images Gateway/Broker `linux/amd64` et `linux/arm64`, bundle Docker Compose, checksums SHA-256 et notes. La CI sécurité produit rapports et SBOM CycloneDX.

## Mise à jour

`scripts/update.sh` choisit par défaut le dernier tag stable de `origin`, refuse le non-fast-forward et conserve le backup.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
