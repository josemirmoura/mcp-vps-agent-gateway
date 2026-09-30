# Construire ou adopter

Vérifié : 2026-09-26.

Avant de construire un composant majeur, évaluez l’existant.

## Gate -1

Demandez :

1. Un produit existant se connecte-t-il au vrai client cible ?
2. Supporte-t-il les opérations requises ?
3. Applique-t-il les permissions server-side ?
4. Peut-il être self-hosted ou satisfaire contrôle/confidentialité ?
5. Fournit-il recovery/révocation ?
6. L’adapter coûte-t-il moins qu’un nouveau control plane privilégié ?

Résultats :

- Adopter
- Adapter/fork
- Construire

## Points de comparaison

### VPS Guardian MCP

Serveur MCP orienté VPS avec opérations structurées et contrôles de sécurité, sans shell générique.

À la date vérifiée : serveur Python côté VPS + launcher npm local via SSH/stdIO. Bon benchmark pour opérations typées, surface de mutation réduite, confirmations, diagnostic/rollback et discipline de release.

Il traite un problème de transport différent de l’architecture cible ChatGPT Web distante.

Projet :
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Service MCP distant hébergé pour filesystem/terminal, via Streamable HTTP et OAuth avec agent appareil appairé.

Benchmark pour UX MCP distante, pairing/révocation, OAuth, ergonomie terminal/filesystem et multi-clients.

Son service hébergé n’est pas l’architecture Broker self-hosted décrite ici.

Projet :
https://github.com/desktop-commander/remote-desktop-commander

## Pourquoi construire cette architecture

Si la combinaison requise est :

- frontière VPS sous contrôle propre ;
- ChatGPT Web comme cible ;
- policy Scoped server-side ;
- opérations typées Linux/Docker/systemd ;
- shell contrôlé optionnel ;
- élévation temporaire optionnelle ;
- sémantique explicite de recovery ;
- Broker privilégié contrôlé par l’opérateur.

## Règle

Ne construisez pas un composant seulement parce qu’il figure sur le diagramme north-star.

Construisez uniquement si l’existant ne satisfait pas le besoin et si le gate MVP précédent en démontre la nécessité.
