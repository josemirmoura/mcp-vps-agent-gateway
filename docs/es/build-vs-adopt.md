# Construir o adoptar

Comprobado: 2026-09-26.

Antes de construir un componente grande, evalúa soluciones existentes.

## Gate -1

Pregunta:

1. ¿Existe un producto que conecte al cliente objetivo real?
2. ¿Soporta las operaciones requeridas?
3. ¿Aplica permisos server-side?
4. ¿Puede ser self-hosted o cumplir control/privacidad?
5. ¿Ofrece recuperación/revocación?
6. ¿Adaptarlo cuesta menos que mantener un nuevo control plane privilegiado?

Resultados:

- Adoptar
- Adaptar/fork
- Construir

## Comparaciones útiles

### VPS Guardian MCP

Servidor MCP para VPS con operaciones estructuradas y controles de seguridad, sin shell genérico.

En la fecha comprobada usa servidor Python en la VPS + launcher npm local sobre SSH/stdIO. Es buen benchmark para:

- operaciones VPS tipadas;
- superficie de mutación estrecha;
- confirmaciones;
- diagnóstico/rollback;
- disciplina de releases.

Resuelve un problema de transporte distinto del objetivo ChatGPT Web remoto.

Proyecto:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Servicio MCP remoto hospedado para filesystem/terminal, usando Streamable HTTP y OAuth con agente emparejado.

Benchmark para:

- UX MCP remota;
- pairing/revocación;
- OAuth;
- ergonomía terminal/filesystem;
- multi-cliente.

Su servicio hospedado no es la arquitectura Broker self-hosted descrita aquí.

Proyecto:
https://github.com/desktop-commander/remote-desktop-commander

## Por qué construir

Se justifica si se necesita esta combinación:

- frontera VPS bajo control propio;
- ChatGPT Web;
- policy Scoped server-side;
- operaciones tipadas Linux/Docker/systemd;
- shell controlado opcional;
- elevación temporal opcional;
- semántica explícita de recovery;
- Broker privilegiado propiedad del operador.

## Regla

No construyas algo solo porque aparece en el diagrama north-star.

Construye cuando una solución existente no cumpla el requisito y el gate MVP anterior demuestre que la capacidad es necesaria.
