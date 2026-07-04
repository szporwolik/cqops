---
title: Manual de Usuario de CQOps
description: Guía para instalar, configurar y usar CQOps — un logger de radioaficionado rápido y orientado al terminal
---

> **Nota de traducción:** Esta traducción fue generada con un modelo LLM. Las correcciones son bienvenidas como Pull Request hacia la rama `dev`. Algunos nombres de pantallas, campos, comandos y atajos se mantienen en inglés intencionadamente para coincidir con la interfaz de CQOps.

# Manual de Usuario de CQOps

CQOps es un logger de radioaficionado rápido y orientado al terminal para operadores que quieren registrar QSOs desde el teclado con baja carga del sistema. Está diseñado para el shack, operación portable, estaciones de club, field days y equipos como Raspberry Pi o portátiles antiguos.

CQOps siempre guarda los QSOs localmente primero. Las integraciones basadas en internet son opcionales.

## Contenido

1. [Qué es CQOps](#qué-es-cqops)
2. [Descarga e instalación](#descarga-e-instalación)
3. [Primer inicio](#primer-inicio)
4. [Registrar tu primer QSO](#registrar-tu-primer-qso)
5. [Pantalla principal](#pantalla-principal)
6. [Flujos de trabajo comunes](#flujos-de-trabajo-comunes)
7. [Registro de QSO](#registro-de-qso)
8. [Logbook Editor y ADIF](#logbook-editor-y-adif)
9. [Concursos](#concursos)
10. [Favorites, referencias y planes de banda](#favorites-referencias-y-planes-de-banda)
11. [Integraciones](#integraciones)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuración](#configuración)
14. [Atajos de teclado](#atajos-de-teclado)
15. [Solución de problemas](#solución-de-problemas)
16. [Reporte de errores](#reporte-de-errores)

---

## Qué es CQOps

CQOps está construido alrededor de la entrada rápida de QSOs, el registro local-first y la operación práctica en campo.

### Ideas principales

- **Terminal-first** — optimizado para uso con teclado.
- **Offline-first** — el registro local de QSOs funciona sin internet.
- **Baja carga** — adecuado para sistemas tipo Raspberry Pi, portátiles antiguos y PCs de estaciones compartidas.
- **Diseño portable** — distribuido como un único binario Go.
- **Varios logbooks** — útil para logs personales, portables, de concursos y de club.
- **Varios operadores** — útil para operación hot-seat y estaciones de club compartidas.
- **Varios rigs** — cada preset de rig puede tener su propio backend y configuración de WSJT-X.
- **Integraciones opcionales** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, control de rig, control de rotor, datos solares y CQOps Live en navegador.

El registro local no requiere internet. Las funciones de red se omiten en modo `--offline`.

### Para quién es CQOps

CQOps encaja bien con:

- operadores portables,
- activadores SOTA y POTA,
- estaciones de club,
- equipos de field day,
- operadores que prefieren un flujo de trabajo en terminal,
- estaciones que necesitan cambiar rápidamente entre operadores, logbooks o rigs.

CQOps no pretende sustituir todas las funciones de un logger de escritorio completo ni de una plataforma web de logbook. Se centra en registro rápido desde terminal, operación en campo, uso offline y flujos de estación compartida.

---

## Descarga e instalación

Todas las versiones:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paquete | Enlace | Notas |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recomendado para la mayoría de usuarios. Añade CQOps al Start Menu y al PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraer y ejecutar sin instalar. |

Usa **Windows Terminal** en lugar de la consola antigua.

### Linux — Debian / Ubuntu

| Arquitectura | Enlace | Uso |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La mayoría de PCs Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemas ARM de 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS de 32 bits |

Instalar el paquete descargado:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — tarball portable

| Arquitectura | Enlace | Uso |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La mayoría de PCs Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemas ARM de 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS de 32 bits |

### macOS

| Arquitectura | Enlace | Uso |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Macs M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Macs Intel |

Instalación manual:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Compilar desde el código fuente

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Compilar desde fuente requiere Go 1.26 o más reciente.

### Requisitos del terminal

| Requisito | Valor |
|---|---|
| Tamaño mínimo del terminal | 80×24 caracteres |
| Tamaño recomendado | 80×43 caracteres o más |
| Terminal recomendado en Windows | Windows Terminal |

### Comandos básicos

```bash
cqops              # Iniciar la TUI
cqops --offline    # Iniciar sin actividad de red
cqops --version    # Mostrar la versión y salir
cqops --help       # Mostrar ayuda
```

---

## Primer inicio

En el primer inicio, CQOps abre el setup wizard. Para el registro local solo se requieren los datos esenciales de la estación. Las integraciones de red pueden omitirse y configurarse después.

### Páginas del wizard

| Página | Qué configura |
|---|---|
| Station & Logbook | Logbook inicial, indicativo de estación, operador, grid locator, referencias y zonas opcionales, Wavelog URL/API/station profile ID |
| Rig | Preset de rig, modelo, antena, potencia, backend, rotor opcional, ajustes UDP opcionales de WSJT-X |
| Integrations | Ajustes de QRZ.com lookup |
| General | Zona horaria IANA |
| Summary | Revisar y guardar |

Backends de rig soportados:

- None,
- flrig,
- Hamlib `rigctld`.

### Navegación en el wizard

| Tecla | Acción |
|---|---|
| Ctrl+S | Validar y continuar; en Summary guarda e inicia CQOps |
| Esc | Volver |
| F10 | Salir |
| Tab / Shift+Tab | Moverse entre campos |
| Space | Cambiar checkbox |

Los ajustes del wizard pueden cambiarse después con **F9**.

---

## Registrar tu primer QSO

1. Inicia CQOps:

   ```bash
   cqops
   ```

2. Completa el setup wizard al menos con tu indicativo y grid locator.
3. Abre el QSO form con **F1**.
4. Introduce el indicativo del contacto. CQOps convierte los indicativos a mayúsculas automáticamente.
5. Completa los demás campos. Si el rig activo está conectado mediante flrig o Hamlib, CQOps puede completar automáticamente frecuencia, banda, mode y submode.
6. Pulsa **Enter** o **Ctrl+S** para guardar.
7. Si aparece una advertencia **DUPE!**, pulsa **Enter** otra vez para guardar igualmente, o **Esc** para cancelar.

El QSO guardado aparece inmediatamente en la tabla Recent QSOs.

---

## Pantalla principal

CQOps usa un diseño fijo de terminal:

```text
┌─ Status Bar ───────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.9  Log Portable  Rig FTDx10  Call SP9MOA/P                         │
│  Net WSJT Hamlib DXC WL                                           23:00L 2100Z │
├─ Tab Bar ──────────────────────────────────────────────────────────────────────┤
│  F1 QSO   F2 QRZ   F4 DXC   F5 HRD   F6 REF   F7 BPL   F8 LOG   F9 CFG         │
├─ Main Content Area ────────────────────────────────────────────────────────────┤
│  QSO form, partner view, map, editor, dashboard data, or active screen content  │
├─ Help Bar ─────────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                              │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Status bar

La Status bar muestra:

- versión de CQOps,
- logbook activo,
- rig activo,
- indicativo de estación,
- operador activo,
- etiquetas de estado de integraciones,
- hora local marcada como `L`,
- hora UTC marcada como `Z`.

Etiquetas comunes: **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** y **WL**.

| Color | Significado |
|---|---|
| Blanco/predeterminado | Conectado o activo |
| Amarillo | Deshabilitado, conectando o offline esperado |
| Rojo | Error o desconectado |
| Acento + negrita | WSJT-X está transmitiendo |

### Pestañas principales

| Tecla | Pestaña | Pantalla |
|---|---|---|
| F1 | QSO | QSO form y Recent QSOs |
| F2 | QRZ | Partner view: datos de callbook, mapa, estadísticas, foto |
| F4 | DXC | Spots de DX Cluster y filtros |
| F5 | HRD | Spots de PSK Reporter y mapa de propagación |
| F6 | REF | Búsqueda de referencias SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, sincronización Wavelog |
| F9 | CFG | Menús de configuración |

La Help bar muestra atajos relevantes para la pantalla activa. **?** abre la ayuda completa.

---

## Flujos de trabajo comunes

### Operación portable, SOTA o POTA

Antes de salir:

1. Ejecuta CQOps una vez con acceso a internet.
2. Permite que CQOps descargue o actualice datos en caché como datos solares, datos REF y prefijos DXCC.
3. Verifica que el Solar panel muestre datos.
4. Verifica que REF search en **F6** devuelva resultados.

En campo:

1. Inicia CQOps en modo offline:

   ```bash
   cqops --offline
   ```

2. Registra normalmente. Los QSOs se guardan localmente.
3. Al volver a estar online, abre **F8** y pulsa **w** para subir los QSOs no enviados a Wavelog.

### Estación de club compartida y hot-seat logging

1. Abre **F9 → Operators**.
2. Pulsa **Ins** para añadir perfiles de operador.
3. En el QSO form, pulsa **Ctrl+O** para cambiar el operador activo.
4. Verifica el operador activo en la Status bar antes de guardar.
5. Usa **Retain** cuando varios operadores necesiten registrar contactos similares sin volver a escribir todo el formulario.

El operador activo se guarda en el campo ADIF `OPERATOR`.

### Logbooks personales y de club

1. Abre **F9 → Logbooks**.
2. Pulsa **Ins** para crear cada logbook.
3. En el QSO form, pulsa **Ctrl+L** para cambiar el logbook activo.
4. Verifica el logbook activo en la Status bar antes de guardar.

Cada logbook puede mantener sus propios datos de estación, ajustes de Wavelog, ajustes de concurso y operadores.

### Varios rigs

1. Abre **F9 → Rigs**.
2. Pulsa **Ins** para crear presets de rig.
3. Selecciona backend: None, flrig o Hamlib.
4. En el QSO form, pulsa **Ctrl+R** para cambiar el rig activo.

Un preset de rig puede incluir backend, modelo, antena, potencia, ajustes de rotor y ajustes UDP de WSJT-X.

### Operación digital con WSJT-X

Cuando la integración UDP de WSJT-X está habilitada, CQOps puede recibir mensajes ADIF de WSJT-X y registrar automáticamente QSOs digitales completados.

Los QSOs auto-registrados:

- se guardan en el logbook activo,
- aparecen inmediatamente en Recent QSOs,
- omiten duplicados,
- heredan el contest ID activo,
- pueden subirse automáticamente a Wavelog cuando Wavelog está configurado y accesible.

Si el operador reportado por WSJT-X no coincide con el operador activo en CQOps, CQOps muestra una advertencia.

Antes de sesiones digitales largas, revisa:

- logbook activo,
- operador activo,
- concurso activo,
- etiqueta de estado de WSJT-X.

### Sincronización Wavelog

CQOps guarda los QSOs localmente primero. La sincronización con Wavelog es opcional.

| Acción | Dónde | Atajo | Notas |
|---|---|---|---|
| Subir QSOs no enviados | Logbook Editor | `w` | Sube en lotes de 50 |
| Descargar desde Wavelog | Logbook Editor | `Ctrl+W` | Descarga incremental usando `last_fetched_id` |

El estado de subida se registra por QSO:

- not sent,
- sent,
- error.

Si la subida falla, el QSO permanece en el logbook local y puede reintentarse más tarde. Purging un logbook reinicia el fetch ID a `0`, permitiendo una descarga completa de nuevo.

---

## Registro de QSO

El QSO form es la pantalla principal de registro. Se abre con **F1**.

CQOps puede completar campos desde:

| Fuente | Campos |
|---|---|
| flrig / Hamlib | Frequency, Freq RX si hay split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | Referencias SOTA, POTA, WWFF, IOTA |
| Wavelog lookup | Worked/confirmed status cuando está configurado |
| DXCC/prefix data | Datos de prefijo y país |

### Diseño del formulario

| Columna izquierda | Columna central | Columna derecha |
|---|---|---|
| Date UTC | Mode | Power W |
| Time UTC | Submode | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band | SIG | SIG Info |
| Exch sent |  |  |
| Exch rcvd |  |  |

Los campos de exchange aparecen solo cuando hay un concurso activo.

La fila inferior contiene:

- **Comment**,
- **Keep** — conserva el campo Comment entre QSOs,
- **Retain** — conserva todo el formulario después de guardar.

Campos como Band, Mode y Submode pueden cambiarse con **PgUp/PgDn**.

### Ruta, rumbo e indicadores

Cuando ambos grid locators son conocidos, CQOps muestra distancia y azimut.

El QSO form también puede mostrar indicadores como:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Guardar

| Tecla | Acción |
|---|---|
| Enter | Guardar QSO |
| Ctrl+S | Guardar QSO desde cualquier campo |
| Esc | Cancelar confirmación de duplicado |
| Enter en confirmación DUPE | Guardar duplicado igualmente |

---

## Logbook Editor y ADIF

Abre el Logbook Editor con **F8**.

Se usa para:

- revisar QSOs,
- edición inline,
- eliminar QSOs,
- importar ADIF,
- exportar ADIF,
- subir a Wavelog,
- descargar desde Wavelog,
- operaciones relacionadas con concursos.

### Editar QSOs

1. Selecciona una fila con **↑/↓**.
2. Pulsa **Enter** o **e**.
3. Edita el QSO.
4. Guarda con **Ctrl+S**.

Los cambios aparecen inmediatamente en Recent QSOs.

### Importar y exportar ADIF

CQOps soporta importación y exportación ADIF 3.1.7.

| Acción | Atajo |
|---|---|
| Importar ADIF | Ctrl+I |
| Exportar ADIF | Ctrl+E |

La importación valida registros, omite duplicados y muestra un resumen. Los QSOs importados se marcan para subida a Wavelog cuando la sincronización Wavelog está configurada.

La exportación puede incluir todos los QSOs o QSOs filtrados por concurso. `CONTEST_ID` se conserva.

### Manejo de modos digitales

El manejo de mode y submode sigue ADIF 3.1.7 según se describe en este manual:

- FT8 se exporta como mode independiente.
- FT4 y FT2 se exportan como MFSK con el submode apropiado.
- Los registros legacy MFSK + FT8 importados se normalizan a FT8 independiente.

El QSO form tiene campos separados **Mode** y **Submode**. Ambos pueden cambiarse con **PgUp/PgDn**.

---

## Concursos

Los concursos añaden campos de exchange y manejo de serial al QSO form.

Crea o configura un concurso en el Logbook Editor con **Ins**.

La configuración de concurso incluye:

- nombre del concurso,
- fecha,
- ADIF contest ID,
- plantillas de exchange.

### Marcadores de plantilla

| Marcador | Se reemplaza por |
|---|---|
| `@rst` | RST enviado o recibido |
| `@serial` | Número serial autoincremental |
| `@call` | Tu indicativo |
| `@grid` | Tu grid locator |
| `@name` | Nombre del operador desde el perfil |

Pulsa **Ctrl+C** para cambiar el concurso activo.

Cuando un concurso está activo:

- el QSO form muestra campos de exchange,
- los números seriales se incrementan automáticamente,
- Recent QSOs puede filtrar QSOs de concurso,
- la exportación ADIF conserva `CONTEST_ID`.

---

## Favorites, referencias y planes de banda

### Favorites

Favorites guarda presets de frecuencia, mode y band en 10 slots.

| Atajo | Acción |
|---|---|
| Alt+0–9 | Recuperar un favorite |
| Alt+Shift+0–9 | Guardar frequency, mode y band actuales como favorite |

Favorites se guardan en la configuración y se comparten entre logbooks.

Ejemplo:

1. Introduce `145.55`.
2. Ajusta mode a `FM`.
3. Ajusta band a `2m`.
4. Pulsa **Alt+Shift+1**.
5. Más tarde, pulsa **Alt+1** para recuperar el preset.

### REF Lookup

Abre REF Lookup con **F6**.

Busca:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Puedes buscar por prefijo, nombre o designador de referencia. Las referencias seleccionadas pueden rellenar el QSO form.

### Band Plan Browser

Abre Band Plan Browser con **F7**.

Da acceso rápido a:

- bandas de radioaficionado,
- rangos VHF/UHF,
- CB,
- PMR446,
- presets de broadcast.

Una frecuencia seleccionada puede usarse para sintonizar el rig activo. Los datos de band plan también pueden exportarse como Markdown.

---

## Integraciones

Todas las integraciones son opcionales. El registro local funciona sin ellas.

### QRZ.com

QRZ.com lookup requiere internet y una suscripción QRZ XML.

En el QSO form, pulsa **Ins** para completar campos de callbook como:

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

La Partner view en **F2** puede mostrar la foto del operador cuando esté disponible.

### Wavelog

La integración Wavelog soporta:

- subida,
- descarga incremental,
- worked/confirmed lookup.

Wavelog se configura por logbook activo con:

- URL,
- API key,
- station profile ID.

CQOps siempre guarda QSOs localmente primero. Un fallo de subida a Wavelog no borra datos locales.

### flrig

La integración flrig usa XML-RPC sobre HTTP.

Endpoint predeterminado:

```text
localhost:12345
```

CQOps puede leer:

- frequency,
- mode,
- power.

En split, VFO A se mapea a Frequency y VFO B a Freq RX.

### Hamlib / rigctld

El control de rig Hamlib usa el daemon TCP `rigctld`.

Según el soporte de la radio y del backend, CQOps puede consultar:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps maneja la falta de soporte para nombres de VFO de forma robusta cuando es posible.

### Hamlib Rotor / rotctld

El control de rotor usa Hamlib `rotctld`.

CQOps soporta:

- azimuth,
- elevation,
- stop commands.

| Atajo | Acción |
|---|---|
| Ctrl+←/→ | Ajustar azimuth en 5° |
| Ctrl+↑/↓ | Ajustar elevation en 5° |
| Ctrl+A | Apuntar rotor al path bearing calculado |
| Ctrl+F1 | Detener rotor |

### WSJT-X

La integración WSJT-X usa mensajes UDP de WSJT-X. CQOps parsea mensajes ADIF y puede registrar automáticamente QSOs completados.

La etiqueta del rig cambia al color de acento mientras WSJT-X transmite. Si el operador reportado por WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

### DX Cluster

La integración DX Cluster usa telnet y requiere internet.

Servidor predeterminado:

```text
dxspots.com:7300
```

Los filtros incluyen:

- band,
- continent,
- mode,
- age/time.

| Tecla | Acción |
|---|---|
| Enter | Rellenar QSO form, sintonizar rig y volver a QSO |
| Space | Sintonizar rig y permanecer en DX Cluster |
| Backspace | Limpiar filtros |

### PSK Reporter

PSK Reporter requiere internet.

Proporciona:

- spots de propagación,
- filtros band/time/mode,
- ASCII world map en **F5**.

### APRS

APRS usa una conexión TCP a un servidor APRS-IS y requiere internet.

Servidor predeterminado:

```text
euro.aprs2.net:14580
```

CQOps puede recibir reportes de posición de estaciones cercanas y mostrarlos en el mapa local de CQOps Live con:

- símbolos estándar,
- popups de callsign,
- auto-fit view,
- range circle configurable.

CQOps también puede enviar un beacon periódico con:

- indicativo de estación,
- SSID,
- grid locator,
- comentario opcional.

APRS se configura por logbook en:

```text
F9 → Logbooks → [active logbook] → APRS
```

### Solar Data

Solar data viene de hamqsl.com e incluye:

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Las actualizaciones live requieren internet. Los datos en caché quedan disponibles offline después de una descarga correcta.

---

## CQOps Live Dashboard

CQOps Live es un dashboard integrado en navegador para actividad de estación en tiempo real.

Es útil para:

- pantallas públicas de field day,
- pantallas de estación de club,
- monitoreo de concursos,
- observar la estación desde otra habitación,
- stands de eventos o ferias.

### Activar el dashboard

1. Pulsa **F9**.
2. Abre **Integrations**.
3. Ve a **HTTP Server**.
4. Habilita **HTTP server**.
5. Opcionalmente configura address y port.
6. Pulsa **Ctrl+S** para guardar.
7. Abre el dashboard en un navegador.

Ajustes predeterminados:

| Ajuste | Predeterminado |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

El servidor arranca inmediatamente después de guardar.

### Modos de visualización

CQOps Live tiene dos modos.

#### Overview mode

Se muestra cuando no se está trabajando un indicativo activo.

Muestra:

- live Leaflet map,
- marcadores QSO de hoy,
- great-circle paths,
- tabla recent QSOs,
- información de estación,
- estadísticas,
- seguimiento de tasa a 5 minutos, 15 minutos y 1 hora,
- top operators,
- QSOs de mayor distancia.

#### Active / Now Working mode

Se muestra cuando se está trabajando un indicativo.

Muestra:

- callsign grande,
- submode indicator,
- foto QRZ si está disponible,
- badges de band y mode,
- indicadores DUPE / NEW CALL / NEW DXCC,
- distance y bearing,
- ruta destacada discontinua en el mapa desde el grid de la estación al grid del corresponsal.

### Info box

La info box encima del local map cambia cada 5 segundos entre módulos:

- band conditions,
- solar activity,
- geomagnetic field,
- último spot de DX Cluster,
- conteos de reportes PSK Reporter por banda.

Band conditions siempre se renderiza a ancho completo.

### Weather row

La weather row muestra condiciones actuales de Open-Meteo para el grid locator de la estación:

- temperature,
- wind,
- humidity,
- icon.

Los datos meteorológicos se obtienen desde el navegador y degradan correctamente cuando no hay conexión.

### Local map

El local map del lado derecho puede mostrar:

- estaciones APRS,
- símbolos APRS estándar,
- range circle,
- popups de callsign,
- day/night terminator overlay opcional,
- RainViewer weather radar overlay opcional.

### Actualizaciones en tiempo real y rendimiento

CQOps Live se actualiza mediante Server-Sent Events (SSE). No hace falta refrescar la página.

El dashboard está diseñado para hardware de baja potencia:

- el navegador renderiza el mapa,
- el navegador calcula distancias,
- el navegador calcula estadísticas,
- CQOps envía actualizaciones JSON ligeras,
- cuando el HTTP server está deshabilitado, no se abre ningún puerto y no se ejecutan goroutines del dashboard.

### Personalización del dashboard

En el formulario de integración HTTP Server puedes configurar:

| Campo | Descripción |
|---|---|
| Header 1 | Título principal en el page header y hero area. Usa “CQOps Live” si está vacío. |
| Header 2 | Subtítulo bajo el título. Usa “Fast, portable ham radio logger” si está vacío. |
| Logo URL | URL pública de imagen mostrada arriba a la izquierda. Usa el logo de CQOps si está vacío. |
| Event Start | Fecha en formato `YYYY-MM-DD`. Filtra estadísticas y listas QSO desde esa fecha. |

---

## Configuración

Abre la configuración con **F9**.

### Archivos de configuración

| Plataforma | Ruta de config |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Las credenciales sensibles se guardan por separado en `secrets.enc` en el mismo directorio de configuración.

Los secrets se cifran con una clave ligada a la máquina. Al mover la configuración a otro equipo, las credenciales deben introducirse de nuevo.

### Menús de configuración

| Menú | Configura |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username y password |
| DX Cluster | Host, port, login |
| Operators | Perfiles de operador |
| Logbooks | Ajustes de station, Wavelog, contest, operator y APRS por logbook |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Usa varios logbooks para operación en casa, portable, concursos y club.

**Ctrl+L** cambia el logbook activo.

Cada logbook mantiene sus propios:

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

Los perfiles de operador contienen:

- indicativo de operador,
- nombre de operador.

**Ctrl+O** cambia el operador activo.

El operador activo se guarda en el campo ADIF `OPERATOR` y acompaña las subidas a Wavelog.

### Multi-rig

Los presets de rig guardan:

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

**Ctrl+R** cambia el rig activo.

### Secrets cifrados

Desde v0.8.7, las credenciales se guardan cifradas.

| Elemento | Valor |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Mismo directorio que `config.yaml` |
| Unix permissions | `0600` donde esté soportado |
| Encryption | AES-256-GCM con clave ligada a la máquina |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

Los plaintext secrets de configuraciones antiguas migran en el primer inicio.

Si `secrets.enc` está corrupto, CQOps arranca con una advertencia y pide reintroducir credenciales.

---

## Atajos de teclado

### Global

| Tecla | Acción |
|---|---|
| F1 | QSO form y Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF Lookup |
| F7 | Band Plan Browser |
| F8 | Logbook Editor |
| F9 | Configuration / main menu |
| F10 | Quit |
| Ctrl+F9 | Log viewer |
| ? | Help overlay |
| Ctrl+L | Cambiar logbook activo |
| Ctrl+R | Cambiar rig activo |
| Ctrl+C | Cambiar concurso activo |
| Ctrl+O | Cambiar operador activo |
| Esc | Volver a la pantalla anterior |

### QSO form

| Tecla | Acción |
|---|---|
| Tab | Campo siguiente |
| Shift+Tab | Campo anterior |
| ↑ / ↓ | Moverse dentro de la columna |
| Enter | Guardar QSO, con confirmación de duplicado si hace falta |
| Ctrl+S | Guardar QSO desde cualquier campo |
| Del | Limpiar todos los campos del formulario |
| Ins | Lookup: QRZ, Wavelog, DXCC y duplicate check |
| PgUp / PgDn | Cambiar band, mode o submode |
| Ctrl+D | Abrir spot dialog |
| Ctrl+T | Cambiar Keep Comment |
| Ctrl+←/→ | Ajustar azimuth del rotor en 5° |
| Ctrl+↑/↓ | Ajustar elevation del rotor en 5° |
| Ctrl+A | Apuntar rotor al bearing desde tu grid al grid del corresponsal |
| Ctrl+F1 | Detener rotor |
| Alt+0–9 | Recuperar favorite |
| Alt+Shift+0–9 | Guardar frequency, mode y band actuales como favorite |

### Logbook Editor

| Tecla | Acción |
|---|---|
| ↑ / ↓ | Navegar filas |
| PgUp / PgDn | Página anterior o siguiente |
| Home / End | Primera o última fila |
| Enter / e | Editar QSO seleccionado |
| Delete | Eliminar QSO seleccionado |
| p | Purge de todos los QSOs |
| Ctrl+C | Cambiar filtro de concurso |
| Ctrl+E | Exportar ADIF |
| Ctrl+I / Tab | Importar ADIF |
| w | Subir QSOs no enviados a Wavelog |
| Ctrl+W | Descargar contactos desde Wavelog |
| Esc / F6 | Cerrar editor y volver al QSO form |

### DX Cluster

| Tecla | Acción |
|---|---|
| ↑ / ↓ | Navegar spots |
| Enter | Rellenar QSO form, sintonizar rig y volver a QSO |
| Space | Sintonizar rig al spot seleccionado y permanecer en DX Cluster |
| Home | Avanzar filtro de band |
| End | Retroceder filtro de band |
| `\` | Cambiar filtro de continent |
| Ins | Avanzar filtro de mode |
| Del | Retroceder filtro de mode |
| PgUp | Avanzar filtro de time |
| PgDn | Retroceder filtro de time |
| Backspace | Limpiar todos los filtros |
| Esc / F4 | Volver al QSO form |

### Partner view

| Tecla | Acción |
|---|---|
| F2 | Cambiar Partner view → Photo → Back |
| Esc / F1 | Volver al QSO form |

---

## Solución de problemas

### CQOps no inicia

Comprueba:

- el terminal tiene al menos 80×24,
- los usuarios de Windows usan Windows Terminal,
- el inicio de red no está bloqueando, probando:

  ```bash
  cqops --offline
  ```

Revisa logs:

| Plataforma | Ruta de logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### El rig no conecta

Para flrig:

- verifica que flrig esté en ejecución,
- verifica el puerto en el preset de rig activo,
- el puerto predeterminado es `12345`.

Para Hamlib:

- verifica que `rigctld` esté en ejecución,
- verifica host y puerto,
- comprueba que tu radio/backend soporte los datos solicitados.

Las etiquetas de estado ayudan al diagnóstico:

| Color | Significado |
|---|---|
| Blanco/predeterminado | Conectado |
| Amarillo | Deshabilitado o conectando |
| Rojo | Fallo |

Los reconnect toasts pueden estar suprimidos. CQOps puede reintentar silenciosamente.

### WSJT-X no registra automáticamente

Comprueba:

- WSJT-X **Settings → Reporting → UDP Server**,
- host y puerto UDP coinciden con el preset de rig activo en CQOps,
- se usa WSJT-X 2.6 o más reciente,
- la etiqueta de estado WSJT está activa,
- el logbook activo es correcto,
- el operador activo es correcto.

### Falla la subida a Wavelog

Comprueba:

- Wavelog URL,
- API key,
- station profile ID,
- etiqueta de estado **WL**.

Los errores de subida se muestran como toasts. Los QSOs permanecen guardados localmente incluso si falla la subida. Un fallo de QSO individual no bloquea el resto del lote.

### Problemas con config file

Config file:

| Plataforma | Ruta |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

El secrets file se guarda en el mismo directorio que `config.yaml`.

Si la configuración está corrupta, muévela o elimínala y reinicia CQOps. El setup wizard creará una configuración nueva.

El campo `last_fetched_id` aparece solo después de una descarga correcta desde Wavelog.

### Problemas de rendimiento

Prueba:

- deshabilitar map rendering en General settings,
- deshabilitar el Solar panel si no es necesario,
- evitar pantallas pesadas de red como DX Cluster y PSK Reporter cuando estés offline,
- usar `cqops --offline` cuando la red no sea fiable.

---

## Reporte de errores

Antes de reportar un error:

1. Habilita **Debug mode** en **F9 → General → Debug**, o define:

   ```yaml
   debug: true
   ```

   en `config.yaml`.

2. Reproduce el problema.
3. Adjunta el log relevante.

Reporta problemas en GitHub:

<https://github.com/szporwolik/cqops/issues>

Incluye:

- versión de CQOps desde `cqops --version`,
- sistema operativo,
- terminal emulator,
- pasos para reproducir,
- debug log relevante.
