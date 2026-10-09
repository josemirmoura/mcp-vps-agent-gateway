# Identidade da versão do runtime

A versão de produto do Pórtico Community tem uma fonte única: o manifesto
`VERSION` no checkout utilizado no build. O pacote Go incorpora esse manifesto
com `go:embed`, sem consultar versões informadas por clientes MCP ou Broker.

| Superfície | Identidade | Fonte |
| --- | --- | --- |
| Handshake MCP `serverInfo.version` | `v<conteúdo de VERSION>` | Gateway `portico.Version()` |
| Ferramenta `system.info.gateway_version` | mesma versão | Gateway local, não resposta do Broker |
| Logs `gateway_start`, `broker_start` | `product_version` | binário compilado |
| Log `portico_cloud_node_start` | `product_version` | binário compilado do conector |
| Protocolo negociado MCP | versão de protocolo, separada | SDK MCP |

Se o manifesto incorporado estiver vazio, a versão anunciada é explicitamente `dev` (nunca apenas `v` ou `v0.1.0`). O formato de release candidate, como `v0.1.0-rc.7`, é preservado.

Uma tag estável **não** pode ser inferida apenas do handshake.
O publicador de releases valida explicitamente que a tag solicitada é
`v` + `VERSION` e que o artefato é construído do commit de release. Build
direto da branch de desenvolvimento incorpora a versão do manifesto, o que
**não** significa que aquela revisão tenha sido publicada como release oficial.
Diagnósticos operacionais devem mostrar também o commit da instalação quando
é importante distinguir modificações posteriores a uma tag.

O teste `TestMCPInitializeAndSystemInfoAdvertiseSamePackagedRelease` abre
sessão MCP HTTP real em servidor de testes, lê `serverInfo.version` da
resposta de inicialização e compara com `system.info.gateway_version` e com
`portico.Version()`. Também prova que valor conflitante retornado pelo Broker
não é usado. `version_test.go` verifica a correspondência com o manifesto.

O conector Cloud permanece um componente diferente da versão de protocolo;
seu log usa a **versão do runtime Community** com campo `product_version`,
sem reivindicar identidade de release ou de versão do serviço SaaS Cloud.
