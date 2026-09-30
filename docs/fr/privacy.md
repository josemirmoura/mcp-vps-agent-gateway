# Confidentialité, données et télémétrie

## Localisation des données

Le paquet est auto-hébergé sur la VPS de l’opérateur. État opérationnel, policy, historique d’audit et identité intégrée y restent sauf export volontaire.

## Audit

Les enregistrements du Broker indiquent qui a invoqué un outil, quelle ressource/action a été demandée, si elle a été autorisée et l’état de séquence/intégrité résultant.

Audit est distinct des logs ordinaires.

## Logs

Les logs Gateway, Broker, Docker et identité peuvent contenir horodatages, identifiants, noms d’outils, erreurs et contexte. Les bundles redigent les secrets configurés et sont testés en CI contre les fuites de tokens, mais restent potentiellement sensibles.

## ChatGPT / MCP

Les données nécessaires à une requête MCP peuvent circuler entre ChatGPT, ou un autre client MCP, et Gateway. La policy contrôle l’autorité server-side. La sortie du modèle ou le contenu distant ne constituent pas une autorisation. N’exposez pas de secrets via fichiers génériques, shell ou résultats d’outils.

## Secrets

Les identifiants locaux sont dans une configuration contrôlée par root/opérateur ou des volumes privés. L’installation ne demande jamais mots de passe VPS, clés SSH privées, root ou secrets sans rapport. L’identifiant OAuth dédié est distinct de VPS/SSH.

## Telemetry

Aucun client analytics/tracking propre. La télémétrie ZITADEL est désactivée avec `ZITADEL_TELEMETRY_ENABLED=false`. Aucun tracking marketing.

## Removal

La suppression sûre conserve configuration, état et audit, retire le runtime actif et renouvelle les identifiants locaux. Le purge complet retire les artefacts MCP et préserve applications, sites, bases, conteneurs tiers, services et fichiers administrés.

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
