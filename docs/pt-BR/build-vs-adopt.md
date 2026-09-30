# Construir ou adotar

Verificado: 2026-09-26.

Antes de construir um componente importante, avalie soluções existentes.

## Gate -1

Pergunte:

1. Existe produto que conecta ao cliente alvo real?
2. Suporta as operações necessárias?
3. Aplica permissões server-side?
4. Pode ser self-hosted ou atender requisitos de controle/privacidade?
5. Oferece recovery/revogação?
6. Adaptá-lo custa menos do que manter um novo control plane privilegiado?

Resultados possíveis:

- Adotar
- Adaptar/forkar
- Construir

## Pontos úteis de comparação

### VPS Guardian MCP

Servidor MCP focado em VPS, publicado, com operações estruturadas e verificações de segurança, sem shell genérico.

Na data verificada, o caminho documentado é servidor Python na VPS + launcher npm local sobre SSH/stdIO. É benchmark forte para:

- operações VPS tipadas;
- superfície de mutação estreita;
- confirmações;
- diagnósticos e rollback;
- disciplina de release.

Resolve um problema de transporte de cliente diferente da arquitetura alvo com ChatGPT Web remoto.

Projeto:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Serviço MCP remoto hospedado para filesystem e terminal, usando Streamable HTTP e OAuth com agente pareado no dispositivo.

É benchmark forte para:

- UX MCP remota;
- pairing/revogação;
- OAuth;
- ergonomia de terminal/filesystem;
- suporte multi-cliente.

Sua implementação hospedada não é a arquitetura Broker self-hosted descrita aqui.

Projeto:
https://github.com/desktop-commander/remote-desktop-commander

## Por que construir esta arquitetura

Construir continua justificado quando a combinação necessária é:

- fronteira VPS sob controle próprio;
- ChatGPT Web como alvo;
- policy Scoped server-side;
- operações tipadas Linux/Docker/systemd;
- shell controlado opcional;
- elevação temporária opcional;
- semântica explícita de recovery;
- Broker privilegiado sob controle do operador.

## Regra

Não construa um componente só porque aparece no diagrama north-star.

Construa apenas quando uma solução existente não satisfizer o requisito e o gate MVP anterior demonstrar que a capacidade é necessária.
