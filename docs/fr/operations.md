# Exploitation

## État

~~~bash
bash scripts/diagnose.sh status
~~~

Affiche version, état Compose, santé Broker/Gateway, redémarrages, snapshot de santé Broker et état de la chaîne d’audit.

## Santé

~~~bash
bash scripts/diagnose.sh health
~~~

Retourne le snapshot de santé du Broker.

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Affiche les logs récents avec redaction des secrets configurés. Ils servent à comprendre redémarrages, échecs d’authentification, perte de liaison Gateway/Broker, timeouts et conteneurs unhealthy.

## Audit

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Répond à : qui, quel outil/ressource/action, allow/deny, résultat, séquence et validité de la chaîne de hash. Logs et audit sont complémentaires.

## Bundle de diagnostic

~~~bash
bash scripts/diagnose.sh bundle
~~~

Contient runtime/version, santé, état/tail audit, logs récents, versions Docker/Compose et snapshot de policy redigé. Le fichier est créé en 0600 ; relisez-le avant partage.

## Version

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

`--check` actualise les tags si possible et indique current/ahead/divergent/update available.

## Mise à jour

~~~bash
bash scripts/update.sh
~~~

La cible par défaut est le dernier SemVer stable. Le script exige un working tree propre, montre les changements, refuse le non-fast-forward, arrête le paquet, sauvegarde `.env`, policy, state et volumes d’identité, teste la migration sur une copie, construit/vérifie la cible et restaure automatiquement en cas d’échec.

RC explicite :

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Suppression sûre

~~~bash
bash scripts/remove.sh safe
~~~

Révoque les autorisations temporaires, supprime le runtime et renouvelle les credentials actifs tout en conservant configuration, policy, audit/state et identité intégrée.

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Le checkout n’est supprimé qu’avec une seconde confirmation :

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Les projets, applications, sites, bases, images/conteneurs tiers, services et fichiers délégués restent intacts.
