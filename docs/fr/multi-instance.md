# Plusieurs instances VPS dans un même workspace ChatGPT

Scénario avancé. L’onboarding normal reste une instance Portico MCP par installation.

## Modèle d’identité

Les noms des tools restent stables entre installations. Ne renommez pas les tools par serveur.

Chaque installation reçoit :

- son endpoint MCP HTTPS ;
- sa relation client/ressource OAuth et ses credentials ;
- un `VPS_AGENT_INSTANCE_ID` stable généré au bootstrap ;
- un `VPS_AGENT_INSTANCE_NAME` visible ;
- l’identité d’instance dans `system.info`, l’audit Broker et les logs structurés.

Exemple :

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

Autre VPS :

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Enregistrez-les comme apps ChatGPT séparées avec noms visibles correspondants. Sélectionnez ou @mentionnez l’app voulue plutôt que de compter sur une désambiguïsation silencieuse.

## Vérification

Pour chaque app, appelez `system.info` et comparez `instance_id`/`instance_name` avec la VPS attendue. Le hostname n’est qu’une information secondaire : un fournisseur peut le réutiliser sur des machines différentes.

Puis inspectez l’audit Broker :

~~~bash
bash scripts/diagnose.sh audit 20
~~~

La même identité doit apparaître dans les nouveaux événements et logs.

## Même conversation

Si ChatGPT autorise plusieurs apps MCP custom dans la même conversation, sélectionnez ou mentionnez explicitement l’app cible avant une opération. Le comportement UI peut changer indépendamment du serveur.

## Gate simultané automatisé

Un workflow dédié lance le vrai paquet Docker simultanément sur trois runners Ubuntu indépendants et réutilise volontairement le même subject, les mêmes tools, path logique et operation id.

Le gate exige :

- IDs/noms d’instance distincts ;
- credentials indépendants ;
- durées d’exécution chevauchantes ;
- même operation id réussissant indépendamment dans les trois stores Broker ;
- chaque audit limité à sa propre identité ;
- hostnames et heads d’audit distincts ;
- état local final attendu sur chaque instance.

C’est le test server-side de collision, sans humain.

## Surface produit ChatGPT

Plusieurs apps live dans le même workspace constituent une expérience UI séparée, non requise pour l’onboarding standard ni le gate E2E single-instance. Si testé, nommez et sélectionnez explicitement chaque VPS.
