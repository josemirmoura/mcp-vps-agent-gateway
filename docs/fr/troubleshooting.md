# Dépannage

Commencez par :

~~~bash
bash scripts/diagnose.sh status
~~~

Puis, si nécessaire :

~~~bash
bash scripts/diagnose.sh bundle
~~~

Relisez le bundle avant partage.

## Répertoire de scope absent

Portico ne crée pas silencieusement des répertoires arbitraires :

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

Ou relancez l’installateur et approuvez l’étape sudo visible.

## Chemin non canonique / symlink

Utilisez un chemin absolu canonique, sans segments point ni ancêtres symlink. C’est un durcissement volontaire de la frontière filesystem.

## Broker/Gateway jamais healthy

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Ne contournez pas la porte de santé.

## Ports 80/443 occupés

Portico ne réutilise un Traefik que s’il peut l’identifier sûrement et ne remplace jamais un serveur web inconnu.

## Plusieurs Traefik

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

Et si le resolver est ambigu :

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS ne résout pas

Corrigez A/AAAA, attendez la propagation et relancez. Le setup valide DNS avant le public.

## Échec de vérification OAuth publique

~~~bash
bash scripts/verify-public.sh
~~~

Vérifiez DNS, certificat, hostname/issuer, metadata protégée, DCR/PKCE OIDC et routes proxy. N’affaiblissez pas issuer/audience.

## ChatGPT ne se connecte pas

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Vérifiez app MCP, OAuth, sélection de Portico et demandez `system.info`. Un timeout n’achève pas l’installation.

## Pas de release stable pour update.sh

Avant le premier stable, aucune cible automatique par design :

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` n’est pas un canal de production automatique.

## Rollback d’update

Ne supprimez pas le backup. Consultez sortie, `backups/<timestamp>/migration-check.json`, `state/update.log` et diagnostic. Le rollback est une protection.

## Secret dans un diagnostic

Ne partagez pas l’artefact. Conservez-le localement, renouvelez les credentials concernés si besoin et suivez `SECURITY.md`.
