# Contrat d’installation

Statut : **décision normative du projet**.

Ce document sépare les procédures temporaires de développement/validation de l’expérience d’installation prise en charge pour l’utilisateur final. L’implémentation, les tests, les README, les tutoriels et la revue de release doivent respecter cette frontière.

## Expérience prise en charge

L’installation officielle est :

- orientée terminal ;
- Docker Compose d’abord ;
- basée sur des commandes et scripts transparents et inspectables ;
- reproductible sur la VPS de l’utilisateur ;
- utilisable sans assistance des développeurs ;
- **natif d’abord** : mécanismes officiels de plateforme, Docker, MCP et OAuth/OIDC standard avant le code de liaison personnalisé.

L’utilisateur installe et exploite Portico directement sur la VPS. ChatGPT n’est connecté qu’une fois le serveur et l’authentification publique prêts.

## Secrets et accès distant

Le tutoriel ne doit **jamais** demander de fournir à ChatGPT ou à un mainteneur :

- le mot de passe VPS ;
- une clé SSH privée ;
- un accès SSH/administratif distant sans restriction ;
- des identifiants root ;
- des identifiants administrateur cloud ;
- une sortie terminal contenant des secrets ;
- tout secret non strictement requis par le service.

Les secrets nécessaires sont saisis localement sur la VPS ou via l’UI/API native du service concerné. Le mot de passe dédié de l’opérateur OAuth est saisi localement et n’est pas fourni à ChatGPT.

Une session de développement peut demander temporairement à un humain d’exécuter des commandes et de retourner des diagnostics nettoyés. C’est une limitation de l’environnement de développement, **pas une exigence produit**.

## Responsabilité d’automatisation

Toute tâche déterministe raisonnablement automatisable doit être absorbée par le paquet : détection des dépendances, ports/proxy, validation Compose, démarrage et santé, HTTPS, bootstrap/discovery OAuth/OIDC, vérification MCP, tests fail-closed, diagnostics avec redaction des secrets et instructions de récupération sûres.

Les commandes d’investigation ponctuelles ne deviennent pas des étapes du tutoriel.

## Actions manuelles délibérées

Elles ne sont admises que si la plateforme impose une décision opérateur ou une action navigateur/UI. La documentation doit dire :

1. quoi faire ;
2. pourquoi cela ne peut pas être automatisé en sécurité ;
3. comment valider le résultat.

Exemples : choisir le plafond physique, configurer le DNS chez un fournisseur externe, saisir localement le mot de passe OAuth et connecter l’app MCP dans ChatGPT Web.

## Porte de finalisation

Le démarrage des conteneurs et la vérification locale sont intermédiaires.

~~~text
VPS configurée
 -> MCP public avec HTTPS valide
 -> OAuth/OIDC fonctionne
 -> connexion ChatGPT Web configurée
 -> appel MCP réel depuis ChatGPT
 -> subject/policy/audit attendus confirmés
 -> INSTALLATION TERMINÉE
~~~

`scripts/connect-chatgpt.sh` fait partie du flux officiel. L’installation n’est terminée que lorsque le Broker observe l’appel ChatGPT authentifié attendu et que la chaîne d’audit reste valide.

## Procédures réservées au développement

Les diagnostics ad hoc, probes Actions temporaires, runners spécifiques, machines CI éphémères, hostnames/IP/paths/branches de développement, curl/openssl d’investigation et SSH développeur appartiennent aux issues/PRs, pas au tutoriel public.

## Revue finale de release

Avant une release :

1. revoir README et traductions officielles ;
2. revoir `installer-flow.md` et `chatgpt-integration.md` dans chaque langue officielle ;
3. revoir la landing page ;
4. retirer les artefacts de développement ;
5. confirmer qu’aucun identifiant VPS/SSH n’est demandé ;
6. documenter raison et validation de chaque action manuelle ;
7. exécuter les checks de contrat d’installation et de couverture i18n ;
8. tester une installation propre ;
9. terminer par un appel MCP réel de ChatGPT et une preuve d’audit Broker.

## Règle d’ingénierie

**NATIF D’ABORD.** Préférer APIs officielles, configuration supportée, Docker/Compose, MCP et OAuth/OIDC. Le code personnalisé doit rester minimal, centralisé, documenté et réversible.
