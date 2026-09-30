# Contrato de instalación

Estado: **decisión normativa del proyecto**.

Este documento define la frontera entre procedimientos temporales de desarrollo/validación y la experiencia de instalación soportada para usuarios finales. Implementación, pruebas, README, tutoriales y revisión de releases deben respetarlo.

## Experiencia soportada

La instalación oficial es:

- orientada al terminal;
- basada primero en Docker Compose;
- construida con comandos y scripts transparentes e inspeccionables;
- reproducible en la VPS del usuario;
- utilizable sin ayuda de los desarrolladores;
- **nativa primero**: mecanismos oficiales de plataforma, Docker, MCP y OAuth/OIDC estándar antes de código auxiliar propio.

El usuario instala y opera Portico directamente en la VPS. ChatGPT se conecta solo después de que el servidor y la autenticación pública estén preparados.

## Secretos y acceso remoto

El tutorial nunca debe pedir al usuario que entregue a ChatGPT o a un mantenedor:

- contraseña de la VPS;
- clave SSH privada;
- SSH irrestricto u otro acceso administrativo remoto;
- credenciales root;
- credenciales administrativas del proveedor cloud;
- salida del terminal con secretos;
- secretos que no sean estrictamente necesarios para el servicio.

Los secretos necesarios se introducen localmente en la VPS o mediante la UI/API nativa correspondiente. La contraseña dedicada del operador OAuth, por ejemplo, se introduce localmente y no se entrega a ChatGPT.

Durante desarrollo, un humano puede ejecutar temporalmente comandos y devolver diagnósticos saneados cuando el agente no tenga un canal de ejecución. Esa es una limitación del entorno de desarrollo, **no un requisito del producto** y nunca debe copiarse al tutorial del usuario.

## Responsabilidad de automatización

Toda tarea determinista que pueda automatizarse razonablemente debe ser absorbida por el paquete, incluyendo:

- detección de dependencias y errores accionables;
- detección de puertos y proxy de borde;
- validación de Compose;
- arranque y health checks;
- configuración/validación HTTPS;
- bootstrap y discovery OAuth/OIDC;
- comprobaciones del endpoint MCP;
- pruebas fail-closed;
- diagnóstico con redacción de secretos;
- instrucciones claras de recuperación cuando no sea seguro decidir automáticamente.

No conviertas comandos de investigación de desarrollo en pasos obligatorios del tutorial.

## Acciones manuales deliberadas

Solo son aceptables cuando la plataforma exige una decisión del operador o una acción en navegador/UI. La documentación debe indicar:

1. qué debe hacer el usuario;
2. por qué no puede automatizarse de forma segura;
3. cómo validar el resultado.

Ejemplos: elegir el techo físico, configurar DNS fuera del control del paquete, introducir localmente la contraseña OAuth y crear/conectar la app MCP en ChatGPT Web.

## Puerta de finalización

Arrancar contenedores y pasar la verificación local son pasos intermedios.

~~~text
VPS configurada
 -> MCP público con HTTPS válido
 -> OAuth/OIDC funcionando
 -> conexión de ChatGPT Web configurada
 -> llamada MCP real desde ChatGPT
 -> subject/policy/auditoría confirmados
 -> INSTALACIÓN COMPLETA
~~~

`scripts/connect-chatgpt.sh` forma parte del flujo oficial. La instalación solo termina cuando el Broker observa la llamada autenticada esperada desde ChatGPT y la cadena de auditoría sigue válida.

## Procedimientos solo de desarrollo

No pertenecen al tutorial soportado salvo promoción explícita:

- pedir al propietario comandos ad hoc y pegar resultados;
- probes temporales en GitHub Actions;
- runners self-hosted específicos;
- máquinas efímeras de CI;
- hostnames, IPs, rutas o branches de una VPS de desarrollo;
- branches temporales;
- probes manuales curl/openssl usados solo para investigar;
- SSH o shell remoto de desarrollador.

Ese material pertenece a issues/PRs o notas de desarrollo.

## Revisión final de release

Antes de declarar una release lista:

1. revisar los README y sus traducciones oficiales;
2. revisar `installer-flow.md` y `chatgpt-integration.md` en cada idioma oficial;
3. revisar la landing pública;
4. eliminar datos específicos del desarrollo;
5. comprobar que nunca se solicitan credenciales VPS/SSH;
6. comprobar que toda acción manual explica motivo y validación;
7. ejecutar el contrato de instalación y la cobertura i18n en CI;
8. instalar desde cero en una VPS soportada;
9. terminar con una llamada real de ChatGPT Web y evidencia de auditoría del Broker.

## Regla de ingeniería

**NATIVO PRIMERO.** Preferir APIs oficiales, configuración soportada, Docker/Compose, MCP, OAuth/OIDC y mecanismos nativos. El código propio solo entra cuando la vía nativa no basta y debe ser mínimo, centralizado, documentado y reversible.
