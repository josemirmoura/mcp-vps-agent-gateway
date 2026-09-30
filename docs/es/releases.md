# Política de releases y versiones

Portico MCP sigue Semantic Versioning.

## Channels

### Estable

Las releases estables usan tags como `v0.1.0`, `v0.1.1` y `v0.2.0`. Son el canal normal de actualización; `main` no es producción.

### Release candidate

Las RC usan tags SemVer de pre-release, como `v0.1.0-rc.1`, para aceptación final.

### Desarrollo

`main` es la rama de desarrollo y no es objetivo predeterminado de `scripts/update.sh`.

## RC5

El candidato actual es `0.1.0-rc.5`. RC5 añade descubrimiento del techo sin abrir contenido, protección de secretos en raíces delegadas, MCP elicitation nativa, anotaciones de seguridad del catálogo público y promoción manual de release vía GitHub. Tags anteriores son historial inmutable.

Stable `v0.1.0` solo se crea después del gate final de aceptación en una instalación limpia.

## Publicación

La ruta preferida es **GitHub Actions → release → Run workflow** en `main`. El workflow valida fuente, pruebas y contrato antes de crear la tag exacta, publicar imágenes multi-arquitectura y crear la GitHub Release.

## Artefactos

Imágenes Gateway/Broker para `linux/amd64` y `linux/arm64`, bundle Docker Compose, checksums SHA-256 y notas. La CI de seguridad produce reportes y SBOM CycloneDX.

## Actualización

Por defecto `scripts/update.sh` elige la tag estable más reciente de `origin`, rechaza non-fast-forward y conserva el backup.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
