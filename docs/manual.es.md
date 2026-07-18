---
title: Manual de usuario de CQOps
description: Guía para instalar, configurar y utilizar CQOps, un registrador de radioaficionado rápido y orientado al terminal
---

# Manual de usuario de CQOps

CQOps es un registrador de radioaficionado rápido y orientado al terminal para operadores que desean registrar contactos de forma fiable con el teclado y con un consumo reducido de recursos. Está diseñado para el uso en la estación, operaciones portátiles, estaciones de club, field days y equipos de la clase Raspberry Pi o portátiles antiguos.

CQOps siempre guarda los QSO primero de forma local. Las integraciones basadas en Internet son opcionales.

## Contenido

1. [Qué es CQOps](#what-cqops-is)
2. [Descarga e instalación](#download-and-installation)
3. [Primer inicio](#first-launch)
4. [Registrar el primer QSO](#log-your-first-qso)
5. [Pantalla principal](#main-screen)
6. [Flujos de trabajo habituales](#common-workflows)
7. [Registro de QSO](#qso-logging)
8. [Editor del libro de guardia y ADIF](#logbook-editor-and-adif)
9. [Concursos](#contests)
    - [Configurar un concurso](#setting-up-a-contest)
    - [Barra de estado inferior](#bottom-status-bar)
    - [Panel de estadísticas del concurso](#contest-statistics-panel)
    - [Exportación ADIF del concurso](#contest-adif-export)
    - [Comportamiento del modo concurso](#contest-mode-behavior)
10. [Favoritos, referencias y planes de banda](#favorites-references-and-band-plans)
11. [Integraciones](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuración](#configuration)
14. [Atajos de teclado](#keyboard-shortcuts)
15. [Solución de problemas](#troubleshooting)
16. [Informar de errores](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## Qué es CQOps

CQOps se basa en la introducción rápida de QSO, el almacenamiento local y una operación práctica en campo.

### Ideas principales

- **Operación orientada al terminal** — optimizada para el uso con teclado.
- **Registro offline-first** — el registro local de QSO funciona sin acceso a Internet. Incluye un mapa mundial incrustado para el dashboard que funciona completamente offline.
- **Bajo consumo** — adecuado para sistemas de la clase Raspberry Pi, portátiles antiguos y ordenadores de estación compartidos.
- **Diseño portátil** — distribuido como un único binario de Go.
- **Varios libros de guardia** — útil para registros personales, portátiles, de concursos y de club.
- **Varios operadores** — útil para flujos de trabajo hot-seat y estaciones de club compartidas.
- **Varios equipos** — cada preset de equipo puede conservar sus propios ajustes de backend y WSJT-X.
- **Integraciones opcionales** — Callbook multiproveedor (QRZ.com, HamQTH, QRZ.RU, Callook.info), Wavelog, DX Cluster, PSK Reporter, GPS, APRS, control del equipo, control del rotor, datos solares y el CQOps Live dashboard para navegador.

El registro local no requiere acceso a Internet. Las funciones de red se omiten en el modo `--offline`.

### Para quién es CQOps

CQOps es una buena opción para:

- operadores portátiles,
- activadores SOTA y POTA,
- estaciones de club,
- equipos de field day,
- operadores que prefieren trabajar desde el terminal,
- estaciones que necesitan cambiar rápidamente entre operadores, libros de guardia o equipos.

CQOps no pretende sustituir todas las funciones de un registrador de escritorio completo ni de una plataforma web de registro. Se centra en el registro rápido desde terminal, la operación en campo, el uso sin conexión y los flujos de trabajo de estaciones compartidas.

### Uso en clubes y estaciones compartidas

CQOps fue diseñado pensando en los clubes de radioaficionados. El operador activo siempre aparece en la barra de estado: **un vistazo** basta para saber quién está utilizando la estación. El cambio de operador requiere una sola combinación (`Ctrl+O`) y se aplica de inmediato; el indicativo y el nombre del operador se escriben en cada QSO posterior. Sin cerrar sesión, sin solicitar contraseña y sin interrupciones.

Los libros de guardia, los presets de equipos y los concursos se cambian de la misma manera: `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Una estación de club con operadores rotativos, varios equipos y varios concursos activos puede cambiar todo el contexto en menos de un segundo sin tocar el ratón.

Durante field days y eventos públicos, el **CQOps Live dashboard** puede mostrar en una pantalla grande un mapa en tiempo real, el flujo de QSO y las estadísticas. Los visitantes y miembros del club pueden seguir la actividad sin amontonarse alrededor del terminal del operador. Basta con activar la integración **HTTP Server** y acceder desde cualquier dispositivo con navegador web.

---

<a id="download-and-installation"></a>
## Descarga e instalación

Ver todas las versiones:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Paquete | Enlace | Notas |
|---|---|---|
| Instalador | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recomendado para la mayoría de usuarios. Añade CQOps al menú Inicio y a PATH. |
| ZIP portátil | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraer y ejecutar sin instalar. |

### Linux — Debian / Ubuntu / Pop!_OS / Linux Mint

Añade el repositorio Cloudsmith APT e instala:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

O descarga el `.deb` directamente:

| Arquitectura | Enlace | Uso |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La mayoría de los PC Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemas ARM de 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS de 32 bits |

Instale el paquete descargado:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Fedora / RHEL / Rocky / AlmaLinux

Añade el repositorio Cloudsmith RPM e instala:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Linux — Arch / Manjaro / CachyOS

Instala desde AUR:

```bash
# CachyOS (usa paru por defecto)
paru -S cqops-bin

# Arch / Manjaro
yay -S cqops-bin
```

También disponible vía `pacaur`, `aura` o `makepkg` manual. PKGBUILD en [aur.archlinux.org/packages/cqops-bin](https://aur.archlinux.org/packages/cqops-bin).

### Linux — archivo Tarball portátil

| Arquitectura | Enlace | Uso |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La mayoría de los PC Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemas ARM de 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS de 32 bits |

### macOS

| Arquitectura | Enlace | Uso |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Mac M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Mac Intel |

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

La compilación desde el código fuente requiere Go 1.26 o posterior.

### Requisitos del terminal

| Requisito | Valor |
|---|---|
| Tamaño mínimo del terminal | 80×24 caracteres |
| Tamaño recomendado | 80×43 caracteres o más |
| Terminal recomendado en Windows | Windows Terminal |
| Terminal con Kitty graphics | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) o [WezTerm](https://wezfurlong.org/wezterm/) |

### Comandos básicos

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

<a id="first-launch"></a>
## Primer inicio

En el primer inicio, CQOps abre el asistente de configuración. Para el registro local solo se requiere la información esencial de la estación. Las integraciones de red se pueden omitir y configurar más adelante.

### Páginas del asistente

| Page | Qué configura |
|---|---|
| Station & Logbook | Libro de guardia inicial, indicativo de estación, operador, grid locator, referencias y zonas opcionales, y Wavelog URL/API/station profile ID |
| Rig | Preset del equipo, modelo, antena, potencia, backend, rotor opcional y ajustes UDP de WSJT-X opcionales |
| Integrations | Configuración de búsqueda del callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info) |
| General | Zona horaria IANA |
| Summary | Revisar y guardar |

Los backends de equipo compatibles son:

- None,
- flrig,
- Hamlib `rigctld`.

### Navegación del asistente

| Key | Action |
|---|---|
| Ctrl+S | Validar y continuar; en **Summary**, guardar e iniciar CQOps |
| Esc | Volver |
| F10 | Salir |
| Tab / Shift+Tab | Moverse entre campos |
| Space | Cambiar casillas de verificación |

Los ajustes del asistente se pueden modificar posteriormente con **F9**.

---

<a id="log-your-first-qso"></a>
## Registrar el primer QSO

1. Inicie CQOps:

   ```bash
   cqops
   ```

2. Complete el asistente de configuración indicando al menos su indicativo y grid locator.

3. Abra **QSO form** con **F1**.

4. Introduzca el indicativo del corresponsal. CQOps convierte automáticamente los indicativos a mayúsculas.

5. Complete los campos restantes. Si el equipo activo está conectado mediante flrig o Hamlib, CQOps puede rellenar automáticamente frequency, band, mode y submode.

6. Pulse **Enter** para guardar.

7. Si aparece la advertencia **DUPE!**, pulse **Enter** de nuevo para guardar de todos modos o **Esc** para cancelar.

El QSO guardado aparece de inmediato en la tabla **Recent QSOs**.

---

<a id="main-screen"></a>
## Pantalla principal

CQOps utiliza un diseño fijo en el terminal:

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

La **Status bar** muestra:

- la versión de CQOps,
- el libro de guardia activo,
- el equipo activo,
- el indicativo de la estación,
- el operador activo,
- las etiquetas de estado de las integraciones,
- la hora local marcada con `L`,
- la hora UTC marcada con `Z`.

Las etiquetas habituales incluyen **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** y **GPS**. La etiqueta **GPS** sigue la misma convención de colores: rojo si está desconectado, amarillo si está conectado pero aún no tiene fix y blanco cuando se ha obtenido una posición.

| Color | Significado |
|---|---|
| Blanco/predeterminado | Conectado o activo |
| Amarillo | Desactivado, conectando o previsiblemente sin conexión |
| Rojo | Error o desconectado |
| Color de acento + negrita | WSJT-X está transmitiendo |

### Pestañas principales

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | **QSO form** y **Recent QSOs** |
| F2 | QRZ | **Partner view**: datos de callbook, mapa, estadísticas, foto |
| F4 | DXC | Spots y filtros de **DX Cluster** |
| F5 | HRD | Spots de **PSK Reporter** y mapa de propagación |
| F6 | REF | Búsqueda de referencias SOTA/POTA/WWFF/IOTA |
| F7 | BPL | **Band Plan Browser** |
| F8 | LOG | **Logbook Editor**, ADIF y sincronización con Wavelog |
| F9 | CFG | Menús de configuración |

La **Help bar** muestra los atajos relevantes para la pantalla activa. Pulse **?** para abrir el **Help overlay** completo.

---

<a id="common-workflows"></a>
## Flujos de trabajo habituales

### Operación portátil, SOTA o POTA

Antes de salir:

1. Ejecute CQOps una vez con acceso a Internet.
2. Permita que CQOps descargue o actualice datos en caché, como datos solares, datos REF y prefijos DXCC.
3. Compruebe que el panel **Solar** muestra datos.
4. Compruebe que la búsqueda **REF** de **F6** devuelve resultados.

En campo:

1. Inicie CQOps en modo sin conexión:

   ```bash
   cqops --offline
   ```

2. Registre los QSO con normalidad. Se guardan localmente.
3. Al volver a tener conexión, abra **F8** y pulse **w** para subir a Wavelog los QSO no enviados.

### Estación de club compartida y registro hot-seat

1. Abra **F9 → Operators**.
2. Pulse **Ins** para añadir perfiles de operador.
3. En **QSO form**, pulse **Ctrl+O** para cambiar el operador activo.
4. Compruebe el operador activo en la barra de estado antes de guardar.
5. Use **Retain** cuando varios operadores necesiten registrar contactos similares sin volver a escribir todo el formulario.

El operador activo se guarda en el campo ADIF `OPERATOR`.

### Libros personales y de club

1. Abra **F9 → Logbooks**.
2. Pulse **Ins** para crear cada libro de guardia.
3. En **QSO form**, pulse **Ctrl+L** para cambiar el libro activo.
4. Compruebe el libro activo en la barra de estado antes de guardar.

Cada libro de guardia puede conservar sus propios datos de estación, ajustes de Wavelog, ajustes de concursos y operadores.

### Varios equipos

1. Abra **F9 → Rigs**.
2. Pulse **Ins** para crear presets de equipo.
3. Seleccione el backend: None, flrig o Hamlib.
4. En **QSO form**, pulse **Ctrl+R** para cambiar el equipo activo.

Un preset de equipo puede incluir backend, modelo, antena, potencia, ajustes del rotor y ajustes UDP de WSJT-X.

### Operación digital con WSJT-X

Cuando la integración UDP de WSJT-X está activada, CQOps puede recibir mensajes ADIF de WSJT-X y registrar automáticamente los QSO digitales finalizados.

Los QSO registrados automáticamente:

- se guardan en el libro activo,
- aparecen inmediatamente en **Recent QSOs**,
- omiten duplicados,
- heredan el contest ID activo,
- se pueden subir automáticamente a Wavelog cuando está configurado y accesible.

Si el operador indicado por WSJT-X no coincide con el operador activo de CQOps, se muestra una advertencia.

Antes de una sesión digital larga, compruebe:

- el libro de guardia activo,
- el operador activo,
- el concurso activo,
- la etiqueta de estado **WSJT**.

### Sincronización con Wavelog

CQOps siempre guarda los QSO primero de forma local. La sincronización con Wavelog es opcional.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | **Logbook Editor** | `w` | Sube en lotes de 50 |
| Download from Wavelog | **Logbook Editor** | `Ctrl+W` | Descarga incremental mediante `last_fetched_id` |

El estado de subida se controla para cada QSO:

- not sent,
- sent,
- error.

Si la subida falla, el QSO permanece en el libro local y se puede reintentar más tarde. Al hacer purge de un libro se restablece la fetch ID a `0`, lo que permite una descarga completa de nuevo.

---

<a id="qso-logging"></a>
## Registro de QSO

**QSO form** es la pantalla principal de registro. Ábrala con **F1**.

CQOps puede rellenar campos desde estas fuentes:

| Fuente | Campos |
|---|---|
| flrig / Hamlib | Frequency, Freq RX si hay split, Mode, Submode |
| Callbook (QRZ.com / HamQTH / QRZ.RU / Callook.info) | Name, QTH, Grid, Country, CQ zone, ITU zone, DXCC, Continent, foto |
| Base de datos REF | Referencias SOTA, POTA, WWFF e IOTA |
| Wavelog lookup | Estado worked/confirmed cuando está configurado |
| Datos DXCC/prefijos | Datos relacionados con el prefijo y el país |

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

Los campos **Exch sent** y **Exch rcvd** solo aparecen cuando hay un concurso activo.

La fila inferior contiene:

- **Comment**,
- **Keep** — conserva el campo **Comment** entre QSO,
- **Retain** — conserva todo el formulario después de guardar.

Los campos **Band**, **Mode** y **Submode** se pueden recorrer con **PgUp/PgDn**.

### Ruta, rumbo e indicadores

Cuando se conocen ambos grid locator, CQOps muestra la distancia y el acimut.

**QSO form** también puede mostrar indicadores como:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Guardado

| Key | Action |
|---|---|
| Enter | Guardar QSO |
| Ctrl+S | Enviar un spot DX desde el formulario completado |
| Esc | Cancelar la confirmación de duplicado |
| Enter en la confirmación DUPE | Guardar el duplicado de todos modos |

---

<a id="logbook-editor-and-adif"></a>
## Editor del libro de guardia y ADIF

Abra **Logbook Editor** con **F8**.

Se utiliza para:

- revisar QSO,
- editar en línea,
- eliminar QSO,
- importar ADIF,
- exportar ADIF,
- subir a Wavelog,
- descargar desde Wavelog,
- realizar operaciones relacionadas con concursos.

### Editar QSO

1. Seleccione una fila con **↑/↓**.
2. Pulse **Enter** o **e**.
3. Edite el QSO.
4. Guarde con **Ctrl+S**.

Los cambios aparecen inmediatamente en **Recent QSOs**.

### Importación y exportación ADIF

CQOps admite importación y exportación ADIF 3.1.7.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

La importación valida los registros, omite duplicados y muestra un resumen. Los QSO importados se marcan para su subida a Wavelog cuando la sincronización está configurada.

La exportación puede incluir todos los QSO o QSO filtrados por concurso. Se conserva `CONTEST_ID`.

### Tratamiento de modos digitales

La gestión de mode y submode sigue ADIF 3.1.7 tal como se describe en este manual:

- FT8 se exporta como mode independiente.
- FT4 y FT2 se exportan como MFSK con el submode correspondiente.
- Los registros antiguos MFSK + FT8 importados se normalizan a FT8 independiente.

**QSO form** tiene campos separados **Mode** y **Submode**. Ambos se pueden recorrer con **PgUp/PgDn**.

---

<a id="contests"></a>
## Concursos

CQOps incluye un panel ligero de registro de concursos diseñado para la **participación ocasional**. No sustituye a registradores específicos como N1MM, Win-Test o TR4W. Para una operación seria multi-op, multi-radio o en categoría assisted, utilice un registrador especializado. CQOps resulta útil cuando desea entregar algunos puntos, seguir su rate por diversión o registrar unos pocos QSO de concurso durante una activación SOTA/POTA sin abandonar su registrador habitual.

<a id="setting-up-a-contest"></a>
### Configurar un concurso

Cree o configure un concurso en **Logbook Editor** con **Ins**.

La configuración incluye:

- nombre del concurso,
- fecha,
- ADIF contest ID,
- plantillas de intercambio.

#### Marcadores de plantilla

| Marker | Replaced with |
|---|---|
| `@rst` | RST enviado o recibido |
| `@serial` | Número de serie que aumenta automáticamente |
| `@cqz` | Zona CQ de la estación DX |
| `@mycqz` | Su zona CQ |
| `@itu` | Zona ITU de la estación DX |
| `@myitu` | Su zona ITU |
| `@grid` | Grid de la estación DX |
| `@mygrid` | Su grid |

Pulse **Ctrl+C** para recorrer los concursos activos o seleccione uno en el menú **Contest** (**F7**). Los campos de intercambio aparecen automáticamente en **QSO form** y los números de serie aumentan de forma automática.

<a id="bottom-status-bar"></a>
### Barra de estado inferior

Cuando hay un concurso activo, la barra inferior muestra un resumen en tiempo real:

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Campo | Significado |
|-------|---------|
| `IARU-HF` | ADIF ID del concurso, es decir, su identificador legible por máquina |
| `· IARU HF` | Nombre visible del concurso, mostrado si difiere del ID |
| `45 QSOs` | Número total de QSO registrados en esta sesión de concurso |
| `Started 16:13` | Hora del primer QSO del concurso en el día actual |
| `Last 14:04 ago` | Tiempo desde el QSO de concurso más reciente |
| `Next #45` | Número de serie que se enviará en el siguiente QSO |
| `On 2:41` | Tiempo total en el aire: suma de los intervalos entre QSO inferiores a 30 minutos |

El campo `Started` se oculta en terminales de menos de 120 columnas. El nombre del concurso y el tiempo `On` se ocultan por debajo de 100 columnas.

<a id="contest-statistics-panel"></a>
### Panel de estadísticas del concurso

Cuando hay un concurso activo y el terminal es suficientemente ancho, aparece a la derecha de **QSO form** un panel compacto con borde amarillo:

```
╭──────────────────────────────────╮
│  Rate     2/h   --/h             │
│  Count 60m   0  hr   0           │
│  Peak  1m120 10m 54 60m 29       │
│  Avg      8/h  Sess 5:36         │
│  QSO/min  last 60m  max 1        │
│                                  │
│                                  │
│                                  │
│                                  │
│  -60m                       now  │
╰──────────────────────────────────╯
```

| Fila | Campo | Significado |
|-----|-------|---------|
| **Rate** | `2/h` | Rate de los últimos **10 QSO**, velocidad de ráfaga a corto plazo |
| | `--/h` | Rate de los últimos **100 QSO**; muestra `--` hasta alcanzar 100 QSO |
| **Count** | `60m 0` | QSO registrados en los últimos 60 minutos |
| | `hr 0` | QSO registrados en la hora de reloj actual desde `:00` |
| **Peak** | `1m120` | Mejor rate de 1 minuto: 120/h significa 2 QSO en ese minuto |
| | `10m 54` | Mejor ventana móvil de 10 minutos: media de 54/h |
| | `60m 29` | Mejor ventana móvil de 60 minutos: media de 29/h |
| **Avg** | `8/h` | Promedio de la sesión: QSO totales divididos por las horas desde el primer QSO |
| | `Sess 5:36` | Duración total de la sesión desde el primer QSO hasta el último, en H:MM o solo minutos |
| **Chart** | `max 1` | El minuto más activo tuvo 1 QSO. Las barras muestran QSO por minuto |
| | `-60m…now` | Borde izquierdo = hace 60 minutos; borde derecho = ahora |

El gráfico utiliza bloques Unicode (`█`) escalados a cuatro filas de barras verticales. Los rates **Peak** omiten el sufijo `/h` porque **Peak** ya implica «por hora». Las duraciones omiten los segundos, que serían ruido con una actualización por minuto.

<a id="contest-adif-export"></a>
### Exportación ADIF del concurso

Para enviar el registro del concurso, abra **Logbook Editor** (`Ctrl+E`) mientras haya un concurso activo. Si se aplica un filtro de concurso, el cuadro de exportación ADIF ofrece exportar **solo los QSO del concurso activo**. Esto genera un archivo ADIF 3.1.7 conforme al estándar, con campos de intercambio, números de serie y el ADIF ID del concurso conservado, listo para subir al robot del organizador o al sistema de comprobación de logs.

<a id="contest-mode-behavior"></a>
### Comportamiento del modo concurso

Cuando hay un concurso activo:

- **QSO form** muestra los campos de intercambio,
- los números de serie aumentan automáticamente,
- **Recent QSOs** se puede filtrar a los QSO del concurso,
- la exportación ADIF conserva `CONTEST_ID`,
- **QSO form**, el panel del concurso y el panel **Solar** reciben un borde amarillo para distinguirlos,
- los spots DXC se comprueban contra todos los QSO del concurso, no solo los del día, para marcar duplicados.

---

<a id="favorites-references-and-band-plans"></a>
## Favoritos, referencias y planes de banda

### Favorites

**Favorites** almacena presets de frequency, mode y band en tres slots, suficientes para las frecuencias de llamada más utilizadas. Los atajos usan `Alt` para evitar conflictos con las teclas habituales de edición del terminal y funcionar de forma fiable en distintos terminales.

| Shortcut | Action |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Recuperar favorite del slot 1, 2 o 3 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Guardar frequency, mode y band actuales en el slot 1, 2 o 3 |

**Favorites** se guarda en la configuración y se comparte entre los libros de guardia.

Ejemplo:

1. Introduzca `145.55`.
2. Establezca **Mode** en `FM`.
3. Establezca **Band** en `2m`.
4. Pulse **Alt+Shift+Ins** para guardar el preset en el slot 1.
5. Más adelante, pulse **Alt+Ins** para recuperarlo.

### REF Lookup

Abra **REF Lookup** con **F6**.

Busca en:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Puede buscar por prefijo, nombre o designador de referencia. Las referencias seleccionadas pueden rellenar **QSO form**.

### Band Plan Browser

Abra **Band Plan Browser** con **F7**.

Proporciona acceso rápido a:

- Amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- Broadcast presets,
- Portable — frecuencias habituales para operación portátil y en campo, incluidas SOTA, POTA y canales de llamada.

Una frecuencia seleccionada se puede usar para sintonizar el equipo activo. Los datos del plan de banda también se pueden exportar como Markdown.

---

<a id="integrations"></a>
## Integraciones

Todas las integraciones son opcionales. El registro local funciona sin ellas.

### Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info)

CQOps admite múltiples proveedores de callbook con cascada basada en prioridad.
Al pulsar **Ins** en el formulario QSO, los proveedores se consultan en orden
hasta que uno devuelve un resultado:

1. **QRZ.com** — requiere Internet y una suscripción QRZ XML. Datos más completos.
2. **HamQTH** — servicio global gratuito. Buena cobertura, requiere cuenta gratuita.
3. **QRZ.RU** — servicio gratuito centrado en Rusia y países vecinos. Requiere inicio de sesión API. Proporciona nombre, QTH, grid, lat/lon, clase, LoTW/eQSL y foto.
4. **Callook.info** — servicio gratuito centrado en EE.UU. Sin cuenta, consultas FCC rápidas.

Si un proveedor de mayor prioridad falla o está desactivado, se prueba el siguiente.
Cuando **Base call fallback** está activado (predeterminado: sí), CQOps también
prueba el indicativo base (sin prefijo ni sufijo) si el indicativo completo no
devuelve resultados.

Active y configure los proveedores en **F9 → Callbook**.

En **QSO form**, pulse **Ins** para rellenar campos del callbook como:

- Name,
- QTH,
- Grid,
- Country,
- CQ/ITU zones,
- DXCC,
- Continent.

**Partner view** en **F2** puede mostrar la foto del operador cuando esté disponible.

> ⚠️ **Experimental.** La visualización de fotos puede utilizar el protocolo
> Kitty terminal graphics y requiere un terminal compatible: Kitty, Ghostty o
> WezTerm. Actívela en **F9 → General → Kitty Graphics**. Los terminales
> estándar y las sesiones SSH sin reenvío de gráficos utilizarán una imagen
> de glifos como alternativa.

### Wavelog

La integración con Wavelog admite:

- subida,
- descarga incremental,
- búsqueda de estado worked/confirmed.

Wavelog se configura para cada libro de guardia activo con:

- URL,
- API key,
- station profile ID.

CQOps siempre guarda los QSO primero de forma local. Un fallo de subida a Wavelog no elimina los datos locales.

### flrig

La integración con flrig utiliza XML-RPC sobre HTTP.

Endpoint predeterminado:

```text
localhost:12345
```

CQOps puede leer:

- frequency,
- mode,
- power.

En operación split, VFO A se asigna a **Frequency** y VFO B a **Freq RX**.

### Hamlib / rigctld

El control de equipo mediante Hamlib utiliza el daemon TCP `rigctld`.

Según el equipo y la compatibilidad del backend, CQOps puede consultar:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps gestiona de forma segura la falta de compatibilidad con nombres de VFO cuando es posible.

### Hamlib Rotator / rotctld

> ⚠️ **Experimental.** El control del rotor es experimental. Compruebe siempre
> los límites físicos de la antena antes de utilizarlo. Esté preparado para
> detener el movimiento inmediatamente con **Alt+/**. Utilícelo con precaución:
> una configuración incorrecta puede dañar el rotor o la antena.

El control del rotor utiliza Hamlib `rotctld`.

CQOps admite:

- azimuth,
- elevation,
- stop commands.

| Shortcut | Action |
|---|---|
| Alt+, | Ajustar azimuth −5° |
| Alt+. | Ajustar azimuth +5° |
| Alt+; | Ajustar elevation +5° |
| Alt+' | Ajustar elevation −5° |
| Alt+\ | Apuntar el rotor al rumbo de ruta calculado |
| Alt+/ | Detener el rotor |

### WSJT-X

La integración con WSJT-X utiliza mensajes UDP de WSJT-X. CQOps analiza mensajes ADIF y puede registrar automáticamente los QSO finalizados.

La etiqueta del equipo adopta el color de acento mientras WSJT-X está transmitiendo. Si el operador indicado por WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

### GPS

CQOps puede leer la posición de un receptor GPS y utilizarla como grid locator de la estación, una solución ideal para operación portátil, móvil o en campo.

Se admiten dos backends:

- **Serial** — conexión directa a un receptor GPS mediante un puerto serie, como USB-to-serial, un puerto COM integrado o `/dev/ttyUSB0`.
- **GPSD** — conexión a un servidor [gpsd](https://gpsd.io/) mediante TCP, de forma predeterminada `127.0.0.1:2947`. Es útil cuando el GPS se comparte con otras aplicaciones o se accede a él por la red.

El indicador **GPS** de la barra de estado muestra:

| Color | Significado |
|--------|---------|
| Rojo `GPS` | Desconectado / error |
| Amarillo `GPS` | Conectado, todavía sin fix |
| Blanco `GPS` | Fix obtenido, posición fijada |

Cuando se obtiene un fix, el grid locator de la estación se sustituye por la posición derivada del GPS y se marca con `(GPS)` en la línea de estado:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Active **Grid from GPS** en los ajustes de **Station & Logbook** para utilizar el grid del GPS en el registro de QSO, los beacons APRS, el mapa del dashboard y los cálculos de distancia.

**Grid precision** se configura en el menú **Integration** con 10, 8 o 6 caracteres. El valor predeterminado es 10 caracteres, aproximadamente 25 m de precisión. El grid siempre se calcula internamente con precisión completa y se recorta en la capa de uso.

### DX Cluster

La integración **DX Cluster** utiliza telnet y requiere acceso a Internet.

Servidor predeterminado:

```text
dxspots.com:7300
```

Los filtros incluyen:

- band,
- spotter continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | Rellenar **QSO form**, sintonizar el equipo y volver a **QSO** |
| Space | Sintonizar el equipo y permanecer en **DX Cluster** |
| Backspace | Borrar todos los filtros |

Cuando **DX Cluster** está conectado, **QSO form** obtiene dos funciones adicionales:

- **Send a spot** — con el formulario completado, pulse **Ctrl+S** para abrir el cuadro de spot y enviar un spot DX al cluster.
- **Nearest spots** — al sintonizar una frecuencia, aparecen directamente en **QSO form** hasta tres spots próximos, para ver qué hay en la banda sin abandonar la pantalla de registro. Pulse **Ctrl+P** para rellenar el indicativo desde el spot más cercano.

### PSK Reporter

La integración **PSK Reporter** requiere acceso a Internet. Es una herramienta excelente para comprobar rápidamente la propagación real: quién recibe su señal o a quién está recibiendo en una banda determinada en ese momento.

Proporciona:

- spots de propagación,
- filtros band/time/mode,
- mapa mundial ASCII en **F5**.

### APRS

CQOps admite tres tipos de servicio APRS. Elija el que corresponda a su configuración:

| Service | Connection | Internet required |
|---|---|---|
| **APRS-IS** | TCP a un servidor APRS-IS | Sí |
| **KISS** | Puerto serie a un TNC KISS de hardware | No |
| **KISS Server** | TCP a un servidor TNC KISS, por ejemplo Dire Wolf | No, basta la red local |

Seleccione el tipo de servicio en el menú **Integrations**:

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Los tres servicios permiten recibir informes de posición APRS de estaciones próximas y mostrarlos en el mapa local de **CQOps Live** con:

- símbolos APRS estándar,
- ventanas emergentes de callsign,
- ajuste automático de la vista,
- círculo de alcance configurable.

También admiten **periodic position beaconing**. CQOps transmite el grid locator de la estación con el intervalo configurado. Cuando GPS está activo y **Grid from GPS** está habilitado, el beacon utiliza automáticamente la posición derivada del GPS, ideal para operación portátil y móvil.

#### APRS-IS

Se conecta a la red mundial APRS-IS mediante Internet. Requiere:

- un indicativo de radioaficionado válido,
- un APRS-IS passcode generado a partir del indicativo,
- conexión a Internet.

Servidor predeterminado:

```text
euro.aprs2.net:14580
```

APRS-IS se configura globalmente en **F9 → Integrations → APRS**. Callsign, SSID, symbol, comment, beacon interval y range filter de cada libro se configuran en **F9 → Logbooks → [active logbook] → APRS**.

#### KISS (serial)

Se conecta directamente a un TNC KISS de hardware mediante un puerto serie. No se requiere Internet: las tramas APRS se envían y reciben a través de la radio.

Configure serial port, baud rate, data bits, parity, stop bits y DTR/RTS en el menú **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS
```

Al seleccionar **KISS**, aparecen los campos de serie **Port**, **Baud**, **Data bits**, **Parity**, **Stop bits**, **DTR** y **RTS**.

El botón **Test** abre el puerto serie para comprobar que el TNC está accesible.

#### KISS Server (TCP)

Se conecta a un TNC KISS accesible mediante TCP, por ejemplo una instancia de [Dire Wolf](https://github.com/wb2osz/direwolf) en el mismo equipo o en la red local. No se requiere Internet.

Introduzca el host y el puerto en el menú **Integrations**:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Valores predeterminados: `127.0.0.1:8001`

#### Beaconing

Los beacons se envían con el intervalo configurado para cada libro de guardia. El intervalo mínimo es de 1 minuto. El beacon incluye:

- callsign de la estación con SSID,
- grid locator derivado del GPS cuando está disponible,
- symbol APRS,
- comment opcional.

Cuando **GPS** está activo y **Grid from GPS** está habilitado en los ajustes de **Station**, el beacon utiliza automáticamente el grid locator derivado del GPS. No es necesario actualizar manualmente el grid durante el movimiento.

Beacon interval y otros ajustes por libro se configuran en:

```text
F9 → Logbooks → [active logbook] → APRS
```

#### Receiving

Los informes de posición APRS recibidos se almacenan localmente en caché y se muestran en el mapa de **CQOps Live dashboard**. Las estaciones aparecen con sus símbolos APRS y se pueden pulsar para ver detalles. La vista se ajusta automáticamente para mostrar todas las estaciones visibles dentro del alcance configurado.

La recepción APRS es independiente de la transmisión de beacons: se puede recibir sin transmitir y viceversa. Basta con activar APRS en **Integrations** y seleccionar el tipo de servicio.

### Solar Data

Los datos solares proceden de hamqsl.com e incluyen:

- SFI,
- sunspot number,
- A/K indices,
- condiciones por banda.

Las actualizaciones en vivo requieren acceso a Internet. Los datos en caché siguen disponibles sin conexión después de una descarga correcta.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

CQOps Live es un dashboard integrado en el navegador para mostrar la actividad de la estación en tiempo real.

Es útil para:

- pantallas públicas en field days,
- monitores de estaciones de club,
- seguimiento de concursos,
- observar la estación desde otra habitación,
- puestos en eventos o ferias.

### Activar el dashboard

1. Pulse **F9**.
2. Abra **Integrations**.
3. Vaya a **HTTP Server**.
4. Active **HTTP server**.
5. Configure opcionalmente address y port.
6. Pulse **Ctrl+S** para guardar.
7. Abra el dashboard en un navegador.

Ajustes predeterminados:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

El servidor se inicia inmediatamente después de guardar.

> **Address binding:** El valor predeterminado `0.0.0.0` permite acceder al dashboard desde cualquier dispositivo de la red local. Es útil para pantallas de field day, estaciones de club o para supervisar la estación desde otra habitación. Establezca address en `127.0.0.1` para limitar el acceso al equipo local.

### Modos de visualización

CQOps Live dispone de dos modos de visualización.

#### Overview mode

Se muestra cuando no se está trabajando ningún callsign activo.

Muestra:

- **live maps** — marcadores de los QSO del día con rutas de círculo máximo desde el grid de la estación hasta cada corresponsal y un mapa APRS local con las estaciones cercanas,
- tabla de recent QSOs,
- información de la estación,
- estadísticas,
- seguimiento del rate de 5 minutos, 15 minutos y 1 hora,
- mejores operadores,
- QSO de mayor distancia.

#### Active / Now Working mode

Se muestra cuando se está trabajando un callsign.

Muestra:

- callsign grande,
- indicador de submode,
- foto de QRZ cuando está disponible,
- indicadores de band y mode,
- indicadores **DUPE / NEW CALL / NEW DXCC**,
- distancia y rumbo,
- ruta de mapa discontinua y resaltada desde el grid de la estación hasta el grid del corresponsal.

### Info box

La **Info box** situada sobre el mapa local alterna cada 5 segundos entre estos módulos:

- condiciones de banda,
- actividad solar,
- campo geomagnético,
- último spot de DX Cluster,
- cantidad de informes de PSK Reporter por banda.

### Weather row

La **Weather row** muestra las condiciones actuales de Open-Meteo para el grid locator de la estación:

- temperatura,
- viento,
- humedad,
- icono.

Los datos meteorológicos se obtienen en el navegador y se omiten de forma segura cuando no hay conexión.

### Local map

El **local map** de la derecha está dedicado a la **supervisión del entorno APRS**. Puede mostrar:

- estaciones APRS próximas con símbolos estándar,
- ventanas emergentes de callsign al pasar el puntero o hacer clic,
- círculo de alcance configurable,
- overlay opcional del terminador día/noche,
- overlay opcional del radar meteorológico RainViewer.

### Actualizaciones en tiempo real y rendimiento

CQOps Live se actualiza mediante Server-Sent Events (SSE). No es necesario recargar la página.

El dashboard está diseñado para hardware de bajo consumo:

- el navegador renderiza los mapas,
- el navegador calcula las distancias,
- el navegador calcula las estadísticas,
- CQOps envía actualizaciones JSON ligeras,
- cuando **HTTP server** está desactivado, no se abre ningún puerto ni se ejecutan goroutines del dashboard.

### Personalización del dashboard

En el formulario de integración **HTTP Server** puede configurar:

| Field | Description |
|---|---|
| Header 1 | Título principal de la cabecera y del área hero. Usa «CQOps Live» como alternativa. |
| Header 2 | Subtítulo situado bajo el título. Usa «Fast, portable ham radio logger» como alternativa. |
| Logo URL | URL pública de la imagen mostrada en la esquina superior izquierda. Usa el logotipo de CQOps como alternativa. |
| Event Start | Fecha con formato `YYYY-MM-DD`. Filtra estadísticas y listas de QSO desde esa fecha. |

---

<a id="configuration"></a>
## Configuración

Abra la configuración con **F9**.

### Archivos de configuración

| Plataforma | Ruta de configuración |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Las credenciales sensibles se guardan por separado en `secrets.enc`, en el mismo directorio de configuración.

Los secrets se cifran con una clave vinculada al equipo. Al mover la configuración a otro equipo, las credenciales deben introducirse de nuevo.

### Menús de configuración

Pulse **F9** para abrir el menú principal y seleccione:

| Menu | Configures |
|---|---|
| General | Units, timezone, partner map/picture, solar panel, fuentes de SCP/REF, Kitty Graphics, Debug mode |
| Logbooks | Station callsign, grid, references, CQ/ITU zones, IARU region, GPS grid; Wavelog por libro (URL, API key, station profile); APRS por libro (callsign, symbol, beacon, range) |
| Operators | Perfiles de operator callsign y operator name para estaciones multioperador |
| Rigs | Presets de equipo: model, antenna, power, backend (None/flrig/Hamlib), rotor, WSJT-X UDP |
| Contests | Perfiles de concurso: name, date, ADIF contest ID, exchange templates, starting serial number |
| Integration | DX Cluster (host, port, login), HTTP Server del dashboard (address, port, branding), GPS service (serial/GPSD, grid precision) |
| Callbook | Proveedores QRZ.com, HamQTH, QRZ.RU, Callook.info; orden de prioridad, base-call fallback, Wavelog lookup |
| Notifications | QSO saved alerts, Wavelog QSO sent status, dupe beep, error sounds |

### Multi-logbook

Utilice varios libros de guardia para home, portable, contest y club.

Pulse **Ctrl+L** para recorrer los libros activos.

Cada libro conserva sus propios:

- datos de estación,
- ajustes de Wavelog,
- ajustes de concurso,
- ajustes de operador.

### Multi-operator

Los perfiles de operador contienen:

- operator callsign,
- operator name.

Pulse **Ctrl+O** para recorrer los operadores activos.

El operador activo se guarda en el campo ADIF `OPERATOR` y se incluye en las subidas a Wavelog.

### Multi-rig

Los presets de equipo almacenan:

- backend,
- model,
- antenna,
- power,
- ajustes del rotor,
- ajustes de WSJT-X.

Pulse **Ctrl+R** para recorrer los equipos activos.

### Secrets cifrados

Desde la versión v0.8.7, las credenciales se guardan cifradas.

| Elemento | Valor |
|---|---|
| Archivo de secrets | `secrets.enc` |
| Ubicación | Mismo directorio que `config.yaml` |
| Permisos Unix | `0600` cuando sea compatible |
| Cifrado | AES-256-GCM con una clave vinculada al equipo |
| Datos protegidos | QRZ password, DX Cluster login, Wavelog API keys |

Los secrets en texto claro de configuraciones anteriores se migran en el primer inicio.

Si `secrets.enc` está dañado, CQOps se inicia con una advertencia y solicita que se vuelvan a introducir las credenciales.

---

<a id="keyboard-shortcuts"></a>
## Atajos de teclado

### Global

| Key | Action |
|---|---|
| F1 | **QSO form** y **Recent QSOs** |
| F2 | **Partner view** |
| F4 | **DX Cluster** |
| F5 | **PSK Reporter** |
| F6 | **REF Lookup** |
| F7 | **Band Plan Browser** |
| F8 | **Logbook Editor** |
| F9 | **Configuration / main menu** |
| F10 | Quit |
| Ctrl+F9 | **Log viewer** |
| ? | **Help overlay** |
| Ctrl+L | Cycle active logbook |
| Ctrl+R | Cycle active rig |
| Ctrl+C | Cycle active contest |
| Ctrl+O | Cycle active operator |
| Esc | Back to previous screen |

### QSO form

| Key | Action |
|---|---|
| Tab | Next field |
| Shift+Tab | Previous field |
| ↑ / ↓ | Move within column |
| Enter | Save QSO, with duplicate confirmation if needed |
| Del | Clear all form fields |
| Ins | Lookup: Callbook, Wavelog, DXCC, and duplicate check |
| PgUp / PgDn | Cycle band, mode, or submode |
| Ctrl+S | Send DX spot from filled form |
| Ctrl+P | Fill call from nearest DXC spot |
| Ctrl+C | Cycle active contest |
| Alt+, | Adjust rotor azimuth −5° |
| Alt+. | Adjust rotor azimuth +5° |
| Alt+; | Adjust rotor elevation +5° |
| Alt+' | Adjust rotor elevation −5° |
| Alt+\ | Point rotor to bearing from own grid to partner grid |
| Alt+/ | Stop rotor |
| Alt+Ins / Alt+Home / Alt+PgUp | Recall favorite (slot 1/2/3) |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Save frequency, mode, band to favorite |

### Logbook Editor

| Key | Action |
|---|---|
| ↑ / ↓ | Navigate rows |
| PgUp / PgDn | Previous or next page |
| Home / End | First or last row |
| Enter / e | Edit selected QSO |
| Delete | Delete selected QSO |
| p | Purge all QSOs |
| Ctrl+C | Cycle contest filter |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Upload unsent QSOs to Wavelog |
| Ctrl+W | Download contacts from Wavelog |
| Esc / F6 | Close editor and return to QSO form |

### DX Cluster

| Key | Action |
|---|---|
| ↑ / ↓ | Navigate spots |
| Enter | Fill QSO form, tune rig, and return to QSO |
| Space | Tune rig to selected spot and stay on DX Cluster |
| Home | Cycle band filter forward |
| End | Cycle band filter backward |
| `\` | Cycle spotter continent filter |
| Ins | Cycle mode filter forward |
| Del | Cycle mode filter backward |
| PgUp | Cycle time filter forward |
| PgDn | Cycle time filter backward |
| Backspace | Clear all filters |
| Esc / F4 | Return to QSO form |

### Partner view

| Key | Action |
|---|---|
| F2 | Cycle Partner view → Photo → Back |
| Esc / F1 | Return to QSO form |

---

<a id="troubleshooting"></a>
## Solución de problemas

### CQOps no se inicia

Compruebe:

- que el terminal tenga al menos 80×24 caracteres,
- que los usuarios de Windows utilicen Windows Terminal,
- que el inicio de red no esté bloqueando, probando:

  ```bash
  cqops --offline
  ```

Compruebe los logs:

| Plataforma | Ruta de logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### El equipo no se conecta

Para flrig:

- compruebe que flrig esté ejecutándose,
- compruebe el puerto del preset activo,
- el puerto predeterminado es `12345`.

Para Hamlib:

- compruebe que `rigctld` esté ejecutándose,
- compruebe host y port,
- compruebe que el equipo/backend admita los datos solicitados.

Las etiquetas de estado ayudan a diagnosticar el problema:

| Color | Significado |
|---|---|
| Blanco/predeterminado | Conectado |
| Amarillo | Desactivado o conectando |
| Rojo | Fallo |

Los toasts de reconexión pueden estar suprimidos. CQOps puede reintentar silenciosamente.

### WSJT-X no registra automáticamente

Compruebe:

- **WSJT-X Settings → Reporting → UDP Server**,
- que UDP host y port coincidan con el preset de equipo activo en CQOps,
- que se utilice WSJT-X 2.6 o posterior,
- que la etiqueta de estado **WSJT** esté activa,
- que el libro activo sea correcto,
- que el operador activo sea correcto.

### La subida a Wavelog falla

Compruebe:

- Wavelog URL,
- API key,
- station profile ID,
- etiqueta de estado **WL**.

Los errores de subida se muestran como toasts. Los QSO permanecen guardados localmente aunque falle la subida. Los fallos de QSO individuales no bloquean el resto del lote.

### Problemas del archivo de configuración

Archivo de configuración:

| Plataforma | Ruta |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Archivo de secrets:

```text
secrets.enc
```

El archivo de secrets se guarda en el mismo directorio que `config.yaml`.

Si la configuración está dañada, mueva o elimine el archivo y reinicie CQOps. El asistente creará una configuración nueva.

El campo `last_fetched_id` solo aparece después de una descarga correcta desde Wavelog.

### Problemas de rendimiento

Pruebe lo siguiente:

- desactive el renderizado de mapas en **General**,
- desactive el panel **Solar** si no es necesario,
- evite pantallas con mucho tráfico de red, como **DX Cluster** y **PSK Reporter**, cuando esté sin conexión,
- utilice `cqops --offline` cuando la red no sea fiable.

---

<a id="reporting-bugs"></a>
## Informar de errores

Antes de informar de un error:

1. Active **Debug mode** en **F9 → General → Debug** o establezca:

   ```yaml
   debug: true
   ```

   en `config.yaml`.

2. Reproduzca el problema.
3. Adjunte el log pertinente.

Informe de los errores en GitHub:

<https://github.com/szporwolik/cqops/issues>

Incluya:

- versión de CQOps de `cqops --version`,
- sistema operativo,
- emulador de terminal,
- pasos para reproducir el problema,
- debug log pertinente.
