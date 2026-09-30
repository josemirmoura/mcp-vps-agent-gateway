# FAQ

## Whole Host signifie-t-il Full ?

Non. Whole Host place le plafond physique du Broker sur `/`. Full est un bundle de capacités séparé, gouverné par policy/feature gates et désactivé par défaut.

## Full signifie-t-il Internet sans restriction ?

Non. Le réseau est une capacité distincte.

## ChatGPT reçoit-il mon mot de passe VPS ou ma clé SSH ?

Non. L’installation est exécutée localement ; ChatGPT reçoit l’endpoint MCP public et complète OAuth.

## Pourquoi le Broker est-il privilégié ?

systemd du host, Docker et filesystem délégué exigent un composant privilégié de confiance. Broker reste local et autoritatif ; Gateway reste non-root sans Docker socket ni host root.

## Docker est-il la frontière de sécurité ?

Non. Docker gère packaging/lifecycle. L’autorisation server-side du Broker est la frontière effective.

## Télémétrie ?

Aucun client analytics/tracking propre. La télémétrie ZITADEL incluse est désactivée. Voir `privacy.md`.

## Un seul projet ?

Oui, profil **Project**.

## Profil recommandé ?

**Standard** : `/opt` comme plafond physique, aucune racine au départ, approbations explicites ensuite.

## Plusieurs répertoires ?

Oui, dynamiquement sous le plafond ou statiquement pour opérateurs avancés.

## Plusieurs VPS ?

Oui. Chaque installation possède identité, credentials, state et chaîne d’audit propres.

## Autres clients MCP ?

Le cœur utilise MCP Streamable HTTP standard. Le chemin public est validé avec ChatGPT Web ; d’autres clients compatibles peuvent fonctionner s’ils supportent l’authentification requise.

## Quand l’installation est-elle terminée ?

Après un véritable appel client traversant authentification, Gateway, Broker, policy, exécution et audit. La santé des conteneurs seule ne suffit pas.

## Pourquoi update.sh évite main ?

`main` est du développement. La production suit les tags SemVer stables.

## Supprimer Portico sans supprimer mes apps ?

Oui. Safe remove et purge retirent les artefacts Portico et préservent les ressources administrées.
