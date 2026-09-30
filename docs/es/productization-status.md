# Estado de productización — v0.1.0-rc.3

Este documento sigue el checklist de productización pública de la rama release-candidate. Es evidencia/estado, no reemplaza arquitectura ni seguridad canónica.

## Completado

- [x] implementación/estado reconciliados con evidencia E2E real de ChatGPT Web;
- [x] README EN/PT-BR orientado al producto;
- [x] guard de saneamiento del repositorio ampliado;
- [x] nombre, tagline y versión canónicos;
- [x] VERSION SemVer + CHANGELOG;
- [x] instalación guiada transparente por terminal;
- [x] UX Project / Custom / Whole Host;
- [x] resumen de autoridad antes del runtime;
- [x] Whole Host separado de Full;
- [x] red irrestricta separada de Full;
- [x] OAuth/OIDC integrado documentado;
- [x] auditoría y observabilidad separadas;
- [x] status/health/log/audit/bundle;
- [x] redacción del bundle;
- [x] canal estable por defecto;
- [x] check de update con `scripts/version.sh --check`;
- [x] backup, migración y rollback automático;
- [x] safe remove y purge;
- [x] landing productizada;
- [x] privacidad/telemetría;
- [x] matriz de compatibilidad/soporte;
- [x] templates de issue/PR;
- [x] validación tag ↔ VERSION;
- [x] checksum;
- [x] SBOM/provenance de contenedor;
- [x] cobertura OAuth inválido/inactivo/expirado/issuer/audience;
- [x] rechazo explícito de token revocado/inactivo;
- [x] rate limiting OAuth/DCR;
- [x] checker de links;
- [x] aceptación del lifecycle del instalador;
- [x] Full/R5 excluido de claims de producción.

## Gates de validación

- [x] PR abierto;
- [x] CI de referencia verde;
- [x] aceptación auth integrada verde;
- [x] paquete Docker verde;
- [x] lifecycle guiado verde;
- [x] aceptación Scoped efímera verde;
- [x] prueba VPS efímera verde;
- [x] instancias simultáneas independientes verde;
- [x] cross-review final.

## No completado deliberadamente

- [ ] tag/release estable `v0.1.0`;
- [ ] aceptación final clean-install del propietario;
- [ ] evidencia R4 de larga duración;
- [ ] madurez Full/R5;
- [ ] clave criptográfica de firma gestionada por el proyecto.

## Metadatos GitHub

README, Pages, release policy, badges/links y contenido se manejan aquí. Descripción/topics/homepage son metadatos del repositorio y deben revisarse en el pase final; no deben marcarse completos sin evidencia de mutación.
