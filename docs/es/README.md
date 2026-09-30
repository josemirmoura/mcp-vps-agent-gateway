# Mapa de documentación

La documentación canónica vive en `docs/`. Esta carpeta es la traducción oficial al español. Si una traducción llegara a diferir del documento inglés de la misma versión, el inglés sigue siendo la referencia técnica hasta que se corrija la traducción.

## Operador / usuario

Empieza aquí:

1. [quick-start.md](quick-start.md) — ruta más corta para instalar.
2. [installation-contract.md](installation-contract.md) — límites normativos de la instalación soportada.
3. [installer-flow.md](installer-flow.md) — flujo completo de instalación.
4. [product-model.md](product-model.md) — modelo de autoridad Project, Standard y Whole Host.
5. [chatgpt-integration.md](chatgpt-integration.md) — conexión con ChatGPT y puerta de finalización.
6. [authentication.md](authentication.md) — OAuth/OIDC integrado.
7. [operations.md](operations.md) — estado, logs, auditoría, actualización y eliminación.
8. [troubleshooting.md](troubleshooting.md) — fallos frecuentes.
9. [faq.md](faq.md) — preguntas frecuentes.
10. [compatibility.md](compatibility.md) — plataformas validadas y no validadas.
11. [privacy.md](privacy.md) — datos, logs, secretos y telemetría.
12. [releases.md](releases.md) — canales SemVer, RC y estable.
13. [support.md](support.md) — límites de soporte.
14. [operator-acceptance.md](operator-acceptance.md) — aceptación final del operador.

## Desarrollo / seguridad

- [project-status.md](project-status.md)
- [architecture.md](architecture.md)
- [policy-schema.md](policy-schema.md)
- [threat-model.md](threat-model.md)
- [security-hardening-v2.md](security-hardening-v2.md)
- [security-release.md](security-release.md)
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md)
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md)
- [transport-and-aggregation.md](transport-and-aggregation.md)
- [implementation-validation.md](implementation-validation.md)
- [simulation-validation.md](simulation-validation.md)
- [mvp-first.md](mvp-first.md)
- [implementation-runbook.md](implementation-runbook.md)
- [build-vs-adopt.md](build-vs-adopt.md)
- [multi-instance.md](multi-instance.md)
- [productization-status.md](productization-status.md)
- [references.md](references.md)

## Precedencia

En caso de conflicto, usa este orden: `architecture.md`, `installation-contract.md`, `project-status.md`, `policy-schema.md`, `security-hardening-v2.md`, `runtime-semantics-and-recovery.md`, documentos de apoyo y, por último, documentos históricos.

**Arquitectura en una frase:** dos procesos Go, un Gateway MCP sin privilegios y un Broker local privilegiado conectados por socket Unix; el Broker controla autorización, estado y ejecución privilegiada.

**Estrategia de producto:** preservar el runtime Scoped validado, hacer explícita la autoridad, distribuir mediante SemVer estable y no presentar Full/R5 como producción hasta completar su madurez propia.
