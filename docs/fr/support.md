# Politique de support

Portico MCP est un projet open source.

## Ligne supportée

Pendant la phase initiale de productisation :

- la release stable `0.x` la plus récente est la ligne publique supportée ;
- la release candidate actuelle est supportée pour acceptation/tests ;
- `main` est du développement et n’est pas un canal de support stable.

Les correctifs de sécurité peuvent exiger la dernière version patch.

## Obtenir de l’aide

Utilisez une issue GitHub pour les bugs reproductibles, échecs d’installation et problèmes de documentation ne contenant aucun secret.

Avant l’ouverture, collectez si possible un bundle de diagnostic nettoyé :

~~~bash
bash scripts/diagnose.sh bundle
~~~

N’attachez jamais `.env`, des identifiants bruts, des clés SSH privées ou du matériel secret non redigé.

## Problèmes de sécurité

Ne publiez pas de détails d’exploitation dans une issue ordinaire. Suivez `SECURITY.md`.

## Niveau de service

Aucun SLA de temps de réponse ou de disponibilité n’est garanti pour le projet open source.

Tout engagement de support d’une future distribution commerciale devra être documenté séparément et ne doit pas être déduit de ce dépôt.
