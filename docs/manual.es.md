---
title: Manual de usuario de CQOps
description: Guía completa para instalar, configurar y usar CQOps — un logger de radioaficionado rápido y orientado al terminal
---

> **Nota:** Esta traducción fue generada con un modelo LLM. Las correcciones son bienvenidas — por favor envíalas como Pull Request a la rama `dev`.

# Manual de usuario de CQOps

## Tabla de contenidos

1. [Acerca de CQOps](#acerca-de-cqops)
2. [Descarga e Instalación](#descarga-e-instalación)
3. [Primer Inicio — Asistente de Configuración](#primer-inicio--asistente-de-configuración)
4. [Inicio Rápido: Registrar su Primer QSO](#inicio-rápido-registrar-su-primer-qso)
5. [Vista General de la Pantalla Principal](#vista-general-de-la-pantalla-principal)
6. [Flujos de Trabajo Comunes](#flujos-de-trabajo-comunes)
7. [Funciones principales](#funciones-principales)
8. [Integraciones](#integraciones)
9. [Referencia de Configuración](#referencia-de-configuración)
10. [Atajos de teclado](#atajos-de-teclado)
11. [Solución de problemas](#solución-de-problemas)

---

## Acerca de CQOps

CQOps es un logger de radioaficionado rápido y orientado al terminal, pensado para operadores que necesitan velocidad, fiabilidad y bajo consumo de recursos — en el shack, en una cumbre, en un field day o en una estación de club compartida.

**Offline-first.** El registro local de QSO no requiere acceso a internet. Los datos de referencia, los datos solares y los prefijos DXCC almacenados en caché siguen disponibles después de descargarse una vez. Las integraciones de red como Wavelog, QRZ.com, DX Cluster y PSK Reporter requieren conectividad y se omiten en el modo `--offline`.

**Construido para operación en campo.** CQOps está preparado para QRP, es cómodo para SOTA/POTA y funciona bien en rigs de clase Raspberry Pi, portátiles antiguos y sistemas sin entorno de escritorio.

**Listo para estaciones de club.** CQOps admite múltiples logbooks, perfiles de operador y presets de rig. Cambie el logbook activo, el operador activo o el rig activo con una sola pulsación.

**Portátil por diseño.** CQOps es un único binario escrito en Go. No tiene dependencia de CGO ni requiere servicios del sistema.

**Multiplataforma.** Windows, Linux y macOS son soportados en amd64 y arm64.

### Para quién es CQOps

- Operadores portables que necesitan registro rápido por teclado en hardware de bajo consumo.
- Activadores SOTA y POTA que registran offline y cargan después.
- Estaciones de club con múltiples operadores compartiendo la misma estación.
- Rigs de field day usando rigs compartidos o hardware clase Raspberry Pi.
- Operadores que prefieren un flujo de trabajo en terminal antes que una GUI de escritorio.

CQOps no pretende reemplazar loggers de escritorio completos ni plataformas de logbook basadas en web. Se centra en logging rápido en terminal, operación en campo, uso offline y flujos de trabajo de estación compartida.

---

## Descarga e instalación

> [Explorar todas las versiones →](https://github.com/szporwolik/cqops/releases)

### Windows

| Paquete | Enlace | Notas |
|---------|------|-------|
| **Instalador** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recomendado para la mayoría de usuarios. Añade CQOps al Menú Inicio y al PATH. |
| ZIP Portátil | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraer y ejecutar sin instalar. |

### Linux — Debian / Ubuntu

| Arquitectura | Enlace | Para |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | La mayoría de PCs Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemas ARM de 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS de 32 bits |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Tarball Portátil

| Arquitectura | Enlace | Para |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | La mayoría de PCs Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemas ARM de 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS de 32 bits |

### macOS

| Arquitectura | Enlace | Para |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Macs M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Macs Intel |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Desde el código fuente

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Solo construir; el binario se escribe en build/
make install      # Construir e instalar en el sistema
```

Las compilaciones desde fuente requieren Go 1.26 o superior.

### Requisitos

- Tamaño de terminal: mínimo 80×24 caracteres.
- Tamaño de terminal recomendado: 80×43 o mayor.
- Se recomienda un emulador de terminal moderno. En Windows, use Windows Terminal en lugar de la consola heredada.

### Opciones de línea de comandos

```bash
cqops              # Iniciar la TUI
cqops --offline    # Iniciar sin actividad de red
cqops --version    # Mostrar versión y salir
cqops --help       # Mostrar ayuda
```

---

## Primer inicio — asistente de configuración

En el primer inicio, CQOps abre un asistente de configuración para los ajustes esenciales de la estación. Las integraciones de red se pueden omitir; el registro local funciona sin ellas.

1. **Estación y Logbook**   
   Configure el logbook inicial, indicativo de estación, operador y grid locator. Los campos opcionales incluyen referencias SOTA/POTA/WWFF, región IARU, zona CQ/ITU, DXCC y SIG/SIG Info. La configuración de Wavelog también está disponible aquí: URL, clave API, ID de perfil de estación, Update y Test.

2. **Rig**   
   Configure un preset de rig: nombre, modelo, antena, potencia y backend de radio. Los backends admitidos son None, flrig y Hamlib rigctld. Los ajustes opcionales incluyen control de rotor mediante Hamlib e integración UDP WSJT-X.

3. **Integraciones**   
   Configure la búsqueda de callbook QRZ.com: opción de activación, nombre de usuario, contraseña oculta y Test.

4. **General**   
   Seleccione la zona horaria IANA. CQOps detecta la zona horaria del sistema por defecto y también proporciona una lista desplazable.

5. **Resumen**   
   Revise la configuración. Presione **Ctrl+S** para guardar e iniciar CQOps.

**Navegación del asistente:** **Ctrl+S** avanza después de la validación. **Esc** retrocede. **F10** sale. Espacio alterna las casillas de verificación. Tab y Shift+Tab se desplazan entre campos.

Todos los ajustes del asistente se pueden cambiar después desde el menú de configuración con **F9**.

---

## Inicio rápido: registrar su primer QSO

1. **Instalar y ejecutar CQOps.**   
   Descargue el paquete para su plataforma, inicie `cqops` y complete el asistente de configuración con al menos su indicativo y grid locator.

2. **Usar el QSO Form.**   
   El QSO Form se abre en **F1**. Ingrese un indicativo; CQOps lo convierte automáticamente a mayúsculas. Si el rig activo está conectado a través de flrig o Hamlib, la frecuencia, banda, mode y submode se rellenan automáticamente. La fecha y hora se establecen en UTC actual.

3. **Moverse por los campos.**   
   Use **Tab**, **Shift+Tab** y **↑/↓**.

4. **Guardar el QSO.**   
   Presione **Enter** o **Ctrl+S**. Si aparece una advertencia **DUPE!**, presione **Enter** de nuevo para guardar de todos modos, o **Esc** para cancelar.

El nuevo QSO aparece inmediatamente en la tabla de Recent QSOs debajo del formulario.

---

## Vista general de la pantalla principal

```text
┌─ Status Bar ───────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Tab Bar ──────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Main Content Area ────────────────────────────────────────────────────┤
│                                                                                  │
│  QSO Form, tabla, mapa, editor o contenido de pantalla activa              │
│                                                                                  │
├─ Help Bar ─────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Status Bar

Status Bar muestra la versión de CQOps, el logbook activo, el rig activo, el indicativo de estación y el operador activo. A la derecha se muestran las etiquetas de estado de integración y la hora en local (`L`) y UTC (`Z`).

**Colores de etiquetas:**

| Color | Significado |
|-------|-------------|
| Blanco/por defecto | Conectado o activo |
| Amarillo | Deshabilitado, conectando o fuera de línea esperado |
| Rojo | Error o desconectado |
| Acento + negrita | WSJT-X está transmitiendo |

Las etiquetas que pueden aparecer incluyen **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** y **WL**.

### Tab Bar

| Tecla | Pestaña | Pantalla |
|-----|-----|--------|
| F1 | QSO | QSO Form y tabla de Recent QSOs |
| F2 | QRZ | Vista de Partner: datos de callbook, mapa, estadísticas, foto |
| F4 | DXC | Spots de DX Cluster y filtros |
| F5 | HRD | Spots de PSK Reporter y mapa de propagación |
| F6 | REF | Búsqueda de referencias SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, sincronización Wavelog |
| F9 | CFG | Menús de configuración |

### Help Bar

La fila inferior muestra los atajos más relevantes para la pantalla activa. Presione **?** para la superposición completa de ayuda.

---

## Flujos de trabajo comunes

### Operación portable / SOTA / POTA

1. **Antes de salir de casa**, ejecute CQOps una vez con acceso a internet. Esto permite a CQOps poblar cachés como datos solares, datos REF y prefijos DXCC.
2. **Verifique la caché** antes de desconectarse. Compruebe que el panel Solar muestra datos y que la búsqueda REF en **F6** devuelve resultados.
3. **En el campo**, inicie CQOps con `cqops --offline`. La actividad de red se omite, lo que evita retrasos de servicios inalcanzables.
4. **Registre normalmente.** El registro local funciona sin internet.
5. **Suba después.** Cuando vuelva a estar en línea, abra el Logbook Editor con **F8** y presione **w** para cargar QSO no enviados a Wavelog.

### Estación de club compartida y hot-seat

1. **Añadir perfiles de operador:** abra **F9 → Operators**, luego presione **Ins** para cada operador. Ingrese su indicativo y nombre.
2. **Cambiar el operador activo:** en el QSO Form, presione **Ctrl+O**. El operador activo se muestra en la Status Bar y se escribe en el campo `OPERATOR` de los QSO guardados.
3. **Usar registro hot-seat:** el operador A registra un QSO, el operador B presiona **Ctrl+O**, luego registra bajo su propio perfil de operador.
4. **Usar Retain cuando sea necesario:** active **Retain** si varios operadores necesitan registrar el mismo contacto sin reescribir el formulario completo.

Antes de guardar en una estación compartida, verifique el logbook activo y el operador activo en la Status Bar.

### Logbook privado + de club

Muchos operadores mantienen un logbook personal y uno o más logbooks de club.

1. **Crear logbooks:** abra **F9 → Logbooks**, luego presione **Ins** para cada logbook.
2. **Cambiar el logbook activo:** presione **Ctrl+L** en el QSO Form. Status Bar muestra el logbook activo.
3. **Mantener datos de estación separados:** cada logbook puede tener su propio indicativo de estación, configuración Wavelog, configuración de concurso y operadores.
4. **Registro dual rápido:** active **Retain**, guarde el QSO en un logbook, presione **Ctrl+L**, luego guárdelo de nuevo en el otro logbook si corresponde.

### Múltiples rigs

1. **Crear presets de rig:** abra **F9 → Rigs**, luego presione **Ins** para cada rig.
2. **Establecer el backend:** use flrig o Hamlib para rigs controlados por CAT. Use None para rigs sintonizados manualmente.
3. **Cambiar el rig activo:** presione **Ctrl+R** en el QSO Form.
4. **Operar estaciones mixtas:** por ejemplo, use un rig HF controlado por CAT y un rig VHF/UHF manual en la misma sesión.
5. **Configurar WSJT-X por rig:** cada preset de rig puede tener su propia configuración UDP WSJT-X.

Cuando el rig activo tiene control CAT, CQOps puede rellenar frecuencia, banda, mode y submode automáticamente. Para rigs manuales, ingréselos usted mismo.

### FT8 / registro automático con WSJT-X

Cuando WSJT-X está conectado a través de UDP, CQOps puede registrar QSO digitales automáticamente desde mensajes ADIF de WSJT-X.

- Los QSO registrados automáticamente se guardan en el logbook activo.
- Los QSO registrados automáticamente duplicados se omiten.
- Los QSO registrados automáticamente heredan el ID de concurso activo.
- Los QSO aparecen inmediatamente en Recent QSOs.
- Si Wavelog está configurado y accesible, los QSO registrados automáticamente pueden subirse automáticamente.
- Si el operador de WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

Verifique el logbook activo, el operador activo y el concurso activo antes de sesiones digitales largas.

### Sincronización Wavelog

La sincronización Wavelog es opcional. CQOps siempre guarda los QSO localmente primero.

**Carga:** presione **w** en el Logbook Editor (**F8**). CQOps sube QSO no enviados en lotes de 50 y rastrea el estado por QSO: no enviado, enviado o error.

**Descarga:** presione **Ctrl+W** en el Logbook Editor. Las descargas son incrementales. CQOps obtiene QSO más nuevos que el `last_fetched_id` guardado para el logbook activo. Los duplicados se omiten.

Si una carga a Wavelog falla, el QSO permanece en el logbook local y puede reintentarse más tarde. Purgar un logbook restablece el ID de obtención a `0`, permitiendo una descarga completa.

---

## Funciones principales

### Registro de QSO

El QSO Form (**F1**) es la pantalla principal de registro. Utiliza un diseño de tres columnas y puede auto-rellenar campos desde control de rig, QRZ.com, lookup de Wavelog, datos DXCC/prefijos y bases de datos REF.

**Campos del formulario:**

| Columna Izquierda | Columna Central | Columna Derecha |
|-------------|---------------|--------------|
| Date UTC | Mode **(▼)** | Power W |
| Time UTC | Submode **(▼)** | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band **(▼)** | SIG | SIG Info |
| Exch sent ⚠️ | | |
| Exch rcvd ⚠️ | | |

⚠️ Los campos de exchange aparecen solo cuando un concurso está activo. Los campos marcados con **(▼)** se ciclan con **PgUp/PgDn**.

La fila inferior contiene:

- **Comment** (Comentario)
- **Keep** — preserva el campo Comment entre QSOs; alternar con **Ctrl+T**
- **Retain** — preserva todo el formulario después de guardar

La línea de ruta/demora muestra distancia y azimut cuando ambos localizadores de cuadrícula son conocidos. También puede mostrar insignias como **DUPE!**, **New Call!** y **New DXCC!**.

### Fuentes de autocompletado

| Fuente | Campos |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX si split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| Base de datos REF | Referencias SOTA, POTA, WWFF, IOTA |
| Búsqueda Wavelog | Estado worked/confirmed cuando está configurado |

### Logging de concurso

Los concursos añaden campos de exchange y manejo de seriales al QSO Form.

Cree o configure un concurso en el Logbook Editor (**F8**) con **Ins**. Establezca el nombre del concurso, fecha, ID de concurso ADIF y plantillas de exchange.

Marcadores de plantilla soportados:

| Marcador | Reemplazado Por |
|--------|---------------|
| `@rst` | RST enviado o recibido |
| `@serial` | Número de serie auto-incrementable |
| `@call` | Su indicativo |
| `@grid` | Su grid locator |
| `@name` | Nombre del operador desde el perfil de operador |

Presione **Ctrl+C** para ciclar el concurso activo. Cuando un concurso está activo:

- el QSO Form muestra campos de exchange,
- los números de serie se auto-incrementan,
- Recent QSOs puede filtrar a QSO de concurso,
- la exportación ADIF preserva `CONTEST_ID`.

### Logbook Editor

El Logbook Editor (**F8**) se usa para gestión de QSO, importación/exportación ADIF, sincronización Wavelog y operaciones relacionadas con concursos.

**Edición en línea:** seleccione una fila con **↑/↓**, presione **Enter** o **e**, edite el QSO, luego guarde con **Ctrl+S**. Los cambios se reflejan en Recent QSOs inmediatamente.

### Importación y exportación ADIF

CQOps soporta importación y exportación ADIF 3.1.7.

- **Ctrl+I** importa un archivo ADIF, valida registros, omite duplicados y muestra un resumen.
- **Ctrl+E** exporta QSOs. La exportación puede incluir todos los QSOs o QSOs filtrados por concurso.
- Los QSOs importados se marcan para cargarse a Wavelog si la sincronización Wavelog está configurada.

### Favoritos

Los favoritos almacenan presets de frecuencia, modo y banda en 10 slots.

| Atajo | Acción |
|----------|--------|
| Alt+0–9 | Recuperar slot favorito |
| Alt+Shift+0–9 | Guardar frecuencia/modo/banda actual en slot |

Los favoritos se almacenan en la configuración y se comparten entre logbooks.

Ejemplo: para una configuración de llamada SOTA FM polaca, ingrese `145.55`, establezca modo `FM`, banda `2m`, luego presione **Alt+Shift+1**. Después, presione **Alt+1** para recuperarlo.

### Búsqueda REF

La pantalla REF (**F6**) busca referencias SOTA, POTA, WWFF e IOTA. Busque por prefijo, nombre o designador de referencia. Las referencias seleccionadas pueden rellenar el QSO Form.

### Band Plan Browser

El Navegador de Planes de Banda (**F7**) proporciona acceso rápido a bandas de aficionados, rangos VHF/UHF, CB, PMR446 y presets de broadcast. Una frecuencia seleccionada puede usarse para sintonizar el rig activo. Los datos de planes de banda también pueden exportarse como Markdown.

---

## Integraciones

### QRZ.com

La búsqueda QRZ.com requiere acceso a internet y una suscripción QRZ XML.

Presione **Ins** en el QSO Form para rellenar campos de callbook como nombre, QTH, grid, país, zonas CQ/ITU, DXCC y continente. La vista Partner (**F2**) puede mostrar la foto del operador cuando está disponible.

### Wavelog

La integración Wavelog requiere acceso a internet. Soporta carga, descarga incremental y búsqueda worked/confirmed.

Wavelog se configura por logbook activo con URL, clave API e ID de perfil de estación. CQOps siempre guarda los QSOs localmente primero; un fallo de carga a Wavelog no pierde datos.

Consulte [Sincronización Wavelog](#sincronización-wavelog).

### flrig

La integración flrig utiliza XML-RPC sobre HTTP. El endpoint predeterminado es `localhost:12345`.

CQOps puede leer frecuencia, modo y potencia desde flrig. La operación split se mapea como VFO A a Frequency y VFO B a Freq RX.

### Hamlib / rigctld

El control de rig Hamlib utiliza el demonio TCP `rigctld`. CQOps puede consultar frecuencia, modo, VFO, split y potencia según el soporte del rig.

Algunos rigs o backends Hamlib no soportan todas las consultas. CQOps maneja la falta de soporte de nombre VFO de forma segura cuando es posible.

### Hamlib Rotor / rotctld

El control de rotor utiliza Hamlib `rotctld`. CQOps soporta comandos de azimut, elevación y parada.

Atajos útiles:

| Atajo | Acción |
|----------|--------|
| Ctrl+←/→ | Ajustar azimut en 5° |
| Ctrl+↑/↓ | Ajustar elevación en 5° |
| Ctrl+A | Apuntar rotor a la demora de ruta calculada |
| Ctrl+F1 | Detener rotor |

### WSJT-X

La integración WSJT-X utiliza mensajes UDP de WSJT-X. CQOps analiza mensajes ADIF y puede registrar automáticamente QSOs completados.

La etiqueta del rig se vuelve de color acento mientras WSJT-X transmite. Si el operador reportado por WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

Consulte [FT8 / registro automático WSJT-X](#ft8--auto-registro-wsjt-x).

### DX Cluster

La integración DX Cluster utiliza una conexión telnet y requiere acceso a internet. El servidor predeterminado es `dxspots.com:7300`.

Los filtros incluyen banda, continente, modo y antigüedad/tiempo. Presione **Enter** en un spot para rellenar el QSO Form, sintonizar el rig activo y volver a la pantalla QSO. Presione **Space** para sintonizar sin rellenar el formulario. Presione **Backspace** para limpiar filtros.

### PSK Reporter

La integración PSK Reporter requiere acceso a internet. Proporciona spots de propagación, filtros de banda/tiempo/modo y un mapa mundial ASCII en **F5**.

### Datos solares

Los datos solares incluyen SFI, número de manchas solares, índices A/K y condiciones banda por banda de hamqsl.com. Las actualizaciones en vivo requieren acceso a internet. Los datos en caché permanecen disponibles offline después de una obtención exitosa.

### CQOps Live — Panel Web

CQOps Live es un panel web integrado que muestra la actividad de su estación en tiempo real en cualquier navegador — perfecto para exhibiciones de Field Day, pantallas de clubes, monitoreo de concursos o para vigilar la estación desde otra habitación.

**Cómo activarlo**

1. Presione **F9** para abrir el menú principal, luego seleccione **Integrations**.
2. Desplácese hasta la sección **HTTP Server** y marque **Enable HTTP server**.
3. Opcionalmente configure la dirección (predeterminado `0.0.0.0`) y el puerto (predeterminado `8073`).
4. Presione **Ctrl+S** para guardar. El servidor se inicia inmediatamente.
5. Abra `http://localhost:8073` (o la dirección configurada) en cualquier navegador.

**Qué muestra el panel**

El panel tiene dos modos que cambian automáticamente:

- **Modo general** (sin indicativo activo): un mapa Leaflet en vivo con marcadores de QSO del día y rutas de círculo máximo, tabla de QSO recientes, información de la estación, estadísticas, operadores destacados y QSOs de mayor distancia.
- **Modo Activo / Now Working** (indicativo en curso): indicativo prominente, foto QRZ (si disponible), insignias de banda/modo, indicadores DUPE/NEW CALL/NEW DXCC, distancia y rumbo, y una línea discontinua resaltada en el mapa desde su estación hasta la ubicación del corresponsal.

Todos los paneles se actualizan en tiempo real mediante Server-Sent Events (SSE) — sin necesidad de recargar la página.

**Personalización**

En el formulario de integración del servidor HTTP puede configurar:

| Campo | Descripción |
|-------|-------------|
| Header 1 | Título principal mostrado en el encabezado y área hero. Valor por defecto: "CQOps Live". |
| Header 2 | Subtítulo debajo del título. Valor por defecto: "Fast, portable ham radio logger". |
| Logo URL | URL de imagen accesible públicamente mostrada en la esquina superior izquierda. Valor por defecto: logo de CQOps. |
| Event Start | Fecha en formato `YYYY-MM-DD`. Cuando se configura, las estadísticas y listas de QSO se filtran desde esta fecha — útil para eventos de varios días. |

**Rendimiento**

El panel está diseñado para hardware de bajo consumo. El navegador realiza todo el renderizado del mapa, cálculos de distancia y estadísticas. La aplicación de terminal CQOps solo envía actualizaciones JSON ligeras a través de SSE. Cuando el servidor HTTP está desactivado, no hay consumo adicional.

**Casos de uso típicos**

- **Field Day / exhibición pública**: conecte una pantalla grande o proyector para mostrar el mapa en vivo y los QSO recientes.
- **Pantalla informativa del club**: monitor dedicado mostrando la actividad de la estación a los visitantes.
- **Monitoreo remoto**: abra el panel en una tableta o teléfono para ver la actividad desde otra habitación.
- **Stand en ferias / eventos**: configure Header 1/2 y el logo del club para una presentación profesional.

---

## Referencia de configuración

La configuración de CQOps se almacena en:

| Plataforma | Ruta de configuración |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Las credenciales sensibles se almacenan por separado en `secrets.enc` en el mismo directorio de configuración. Los credenciales están cifrados con una clave vinculada a la máquina, por lo que las credenciales deben reingresarse al mover una configuración a otra máquina.

Abra la configuración con **F9**.

| Menú | Configura |
|------|------------|
| Station | Indicativo, grid, zona CQ/ITU, región IARU, referencias |
| Rig | Presets de rig, modelo, antena, potencia, backend, rotor, WSJT-X |
| Wavelog | URL, clave API, ID de perfil de estación |
| QRZ | Nombre de usuario y contraseña |
| DX Cluster | Host, puerto, login |
| Operators | Perfiles de operador: indicativo y nombre |
| Logbooks | Configuración de estación, Wavelog, concurso y operador por logbook |
| Notifications | Comportamiento de toasts y notificaciones |
| General | Zona horaria, unidades de distancia, mapa, modo debug |

### Multi-Logbook

Use múltiples logbooks para operación en casa, portátil, concurso y club. Presione **Ctrl+L** para ciclar el logbook activo. Cada logbook mantiene sus propios detalles de estación, configuración Wavelog, configuración de concursos y operadores.

### Multi-Operador

Los perfiles de operador contienen indicativo y nombre del operador. Presione **Ctrl+O** para ciclar el operador activo. El operador activo se guarda en el campo ADIF `OPERATOR` y se usa en las cargas a Wavelog.

### Multi-Rig

Los presets de rig almacenan backend, modelo, antena, potencia, rotor y configuración WSJT-X. Presione **Ctrl+R** para ciclar el rig activo.

### Credenciales cifradas

Desde v0.8.7, las credenciales se almacenan cifradas.

- **Archivo de credenciales:** `secrets.enc`
- **Ubicación:** mismo directorio que `config.yaml`
- **Permisos Unix:** `0600` donde se soporte
- **Cifrado:** AES-256-GCM con clave vinculada a la máquina
- **Datos protegidos:** contraseña QRZ, login DX Cluster, claves API Wavelog
- **Migración:** credenciales en texto plano de configuraciones antiguas migran en el primer inicio
- **Recuperación:** si `secrets.enc` está corrupto, CQOps inicia con una advertencia y solicita reingresar las credenciales

---

## Atajos de teclado

### Globales

| Tecla | Acción |
|-----|--------|
| F1 | QSO Form y Recent QSOs |
| F2 | Partner View |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | Búsqueda REF |
| F7 | Navegador de Planes de Banda |
| F8 | Logbook Editor |
| F9 | Configuración / menú principal |
| F10 | Salir |
| Ctrl+F9 | Visor de logs |
| ? | Superposición de ayuda |
| Ctrl+L | Ciclar logbook activo |
| Ctrl+R | Ciclar rig activo |
| Ctrl+C | Ciclar concurso activo |
| Ctrl+O | Ciclar operador activo |
| Esc | Volver a pantalla anterior |

### QSO Form — F1

| Tecla | Acción |
|-----|--------|
| Tab | Campo siguiente |
| Shift+Tab | Campo anterior |
| ↑ / ↓ | Moverse dentro de la columna |
| Enter | Guardar QSO, con confirmación de duplicado si es necesario |
| Ctrl+S | Guardar QSO desde cualquier campo |
| Del | Limpiar todos los campos del formulario |
| Ins | Búsqueda: QRZ, Wavelog, DXCC y verificación de duplicado |
| PgUp / PgDn | Ciclar banda, modo o submodo |
| Ctrl+D | Abrir diálogo de spot |
| Ctrl+T | Alternar Keep Comment |
| Ctrl+←/→ | Ajustar azimut del rotor en 5° |
| Ctrl+↑/↓ | Ajustar elevación del rotor en 5° |
| Ctrl+A | Apuntar rotor a demora desde grid propio a grid del partner |
| Ctrl+F1 | Detener rotor |
| Alt+0–9 | Recuperar slot favorito |
| Alt+Shift+0–9 | Guardar frecuencia/modo/banda actual en slot favorito |

### Logbook Editor — F8

| Tecla | Acción |
|-----|--------|
| ↑ / ↓ | Navegar filas |
| PgUp / PgDn | Página anterior o siguiente |
| Home / End | Primera o última fila |
| Enter / e | Editar QSO seleccionado |
| Delete | Eliminar QSO seleccionado |
| p | Purgar todos los QSOs |
| Ctrl+C | Alternar filtro de concurso |
| Ctrl+E | Exportar ADIF |
| Ctrl+I / Tab | Importar ADIF |
| w | Subir QSOs no enviados a Wavelog |
| Ctrl+W | Descargar contactos de Wavelog |
| Esc / F6 | Cerrar editor, volver a QSO |

### DX Cluster — F4

| Tecla | Acción |
|-----|--------|
| ↑ / ↓ | Navegar spots |
| Enter | Rellenar formulario + sintonizar rig + ir a QSO |
| Space | Sintonizar rig al spot (permanecer en DXC) |
| Home | Filtro de banda adelante |
| End | Filtro de banda atrás |
| \\ | Filtro de continente |
| Ins | Filtro de modo adelante |
| Del | Filtro de modo atrás |
| PgUp | Filtro de tiempo adelante |
| PgDn | Filtro de tiempo atrás |
| Backspace | Limpiar todos los filtros |
| Esc / F4 | Volver al QSO Form |

### Partner View — F2

| Tecla | Acción |
|-----|--------|
| F2 | Ciclo: Partner View → Foto → Volver |
| Esc / F1 | Volver al QSO Form |

---

## Solución de problemas

### La aplicación no se inicia

- El terminal debe tener al menos 80×24 caracteres.
- En Windows, use Windows Terminal, no la consola heredada `cmd.exe`.
- Pruebe `cqops --offline` para descartar problemas de red.
- Revise los logs: `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) o `%APPDATA%\cqops\logs\` (Windows).

### El rig no se conecta

- **flrig:** verifique que flrig esté funcionando y el puerto coincida (predeterminado `12345`).
- **Hamlib:** verifique que rigctld esté funcionando y el puerto TCP sea correcto.
- Color de etiqueta de estado: blanco = conectado, amarillo = conectando/deshabilitado, rojo = error.
- Los toasts de reconexión suprimidos son normales — CQOps reintenta en segundo plano.

### WSJT-X no registra automáticamente

- Verifique la configuración UDP de WSJT-X: Settings → Reporting → UDP Server.
- WSJT-X debe ser versión 2.6 o superior.
- La etiqueta de estado debería ser blanca (por defecto) cuando WSJT-X está funcionando.

### La carga a Wavelog falla

- Verifique URL, clave API e ID de perfil de estación en la configuración.
- Etiqueta de estado: blanco = accesible, amarillo = deshabilitado/sin internet, rojo = error.
- Los errores de carga se muestran como toasts; los QSOs permanecen guardados localmente.
- Los fallos individuales de QSO no bloquean el resto del lote.

### Problemas con el archivo de configuración

- Configuración: `~/.config/cqops/config.yaml` (Linux/macOS) o `%APPDATA%\cqops\config.yaml` (Windows).
- credenciales: `secrets.enc` en el mismo directorio.
- Si la configuración está corrupta, elimínela y reinicie — el asistente creará una nueva.
- El campo `last_fetched_id` solo aparece después de una descarga Wavelog exitosa.

### Rendimiento

- Desactive el renderizado de mapa y el panel solar en ajustes General.
- Cierre pestañas no utilizadas (DXC, PSK).
- Ejecute con `--offline` si la red no es confiable.

### Reportar errores

Active el **modo Debug** antes de reproducir un problema — F9 → General → Debug, o establezca `debug: true` en la configuración. Los logs completos se escriben en el directorio de logs específico de la plataforma.

Reporte problemas en [GitHub Issues](https://github.com/szporwolik/cqops/issues) con:
- Versión de CQOps (`cqops --version`)
- Sistema operativo y emulador de terminal
- Pasos para reproducir
- Log de debug
