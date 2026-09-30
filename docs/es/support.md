# Política de soporte

Portico MCP es un proyecto open source.

## Línea soportada

Durante la fase inicial de productización:

- la release estable `0.x` más reciente es la línea pública soportada;
- la release candidate actual se soporta para aceptación/pruebas;
- `main` es desarrollo y no es un canal estable de soporte.

Las correcciones de seguridad pueden exigir actualizar al patch más reciente.

## Obtener ayuda

Usa una issue de GitHub para bugs reproducibles, fallos de instalación y problemas de documentación que no contengan secretos.

Antes de abrirla, genera un bundle de diagnóstico saneado cuando sea práctico:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Nunca adjuntes `.env`, credenciales sin procesar, claves SSH privadas ni material secreto sin redacción.

## Problemas de seguridad

No publiques detalles de explotación en una issue normal. Sigue `SECURITY.md`.

## Nivel de servicio

No existe SLA garantizado de tiempo de respuesta ni disponibilidad para el proyecto open source.

Cualquier compromiso de soporte de una futura distribución comercial deberá documentarse por separado y no debe inferirse de este repositorio.
