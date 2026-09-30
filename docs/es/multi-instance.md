# Múltiples instancias VPS en un mismo workspace de ChatGPT

Este es un escenario avanzado. El onboarding normal sigue siendo una instancia de Portico MCP por instalación.

## Modelo de identidad

Los nombres de las tools permanecen estables entre instalaciones. No cambies los nombres por servidor.

Cada instalación obtiene:

- su propio endpoint MCP HTTPS;
- su propia relación cliente/recurso OAuth y credenciales;
- un `VPS_AGENT_INSTANCE_ID` estable generado en bootstrap;
- un `VPS_AGENT_INSTANCE_NAME` visible para el operador;
- identidad de instancia en `system.info`, auditoría del Broker y logs estructurados.

Ejemplo:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

En otra VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Regístralas como apps separadas en ChatGPT con nombres visibles correspondientes. Selecciona o @menciona la app correcta en vez de depender de una desambiguación silenciosa de tools con nombres iguales.

## Verificación

Para cada app, llama `system.info` y compara `instance_id` y `instance_name` con la VPS esperada. Considera el hostname solo como información de apoyo, porque proveedores pueden reutilizarlo en máquinas desechables distintas.

Después inspecciona la auditoría del Broker:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

El mismo ID/nombre debe aparecer en los nuevos eventos y logs.

## Uso en la misma conversación

Si la interfaz de ChatGPT permite varias apps MCP custom en una conversación, selecciona o menciona explícitamente la app antes de cada operación. La UI puede cambiar independientemente del servidor.

## Gate automatizado simultáneo

El repositorio incluye un workflow de tres máquinas. Inicia el paquete Docker real en tres runners Ubuntu independientes al mismo tiempo y reutiliza deliberadamente el mismo subject, nombres de tools, path lógico y operation id.

Pasa solo si:

- las tres instalaciones tienen IDs/nombres distintos;
- usan credenciales independientes;
- sus tiempos de vida se solapan;
- el mismo operation id funciona de forma independiente en los tres stores del Broker;
- cada auditoría registra solo su identidad;
- hostnames y heads de auditoría siguen distintos;
- cada instancia termina en el estado esperado.

Es una prueba server-side de colisiones sin intervención humana.

## Superficie ChatGPT

Conectar varias apps vivas al mismo workspace es un experimento de UI separado. No es requisito del onboarding normal ni del gate final single-instance. Si se prueba, registra cada VPS como app con nombre propio y selecciona explícitamente la correcta.
