---
title: Manual de Usuario de CQOps
description: Guía completa para instalar, configurar y usar CQOps — un rápido libro de guardia para radioaficionados basado en terminal
---

> **Nota:** Esta traducción fue generada con un modelo LLM. Las correcciones son bienvenidas — por favor envíalas como Pull Request a la rama `dev`.

# Manual de Usuario de CQOps

## Tabla de Contenidos

1. [Acerca de CQOps](#acerca-de-cqops)
2. [Descarga e Instalación](#descarga-e-instalación)
3. [Primer Inicio — Asistente de Configuración](#primer-inicio--asistente-de-configuración)
4. [Inicio Rápido: Registrar su Primer QSO](#inicio-rápido-registrar-su-primer-qso)
5. [Vista General de la Pantalla Principal](#vista-general-de-la-pantalla-principal)
6. [Flujos de Trabajo Comunes](#flujos-de-trabajo-comunes)
7. [Funciones Principales](#funciones-principales)
8. [Integraciones](#integraciones)
9. [Referencia de Configuración](#referencia-de-configuración)
10. [Atajos de Teclado](#atajos-de-teclado)
11. [Solución de Problemas](#solución-de-problemas)

---

## Acerca de CQOps

CQOps es un rápido libro de guardia para radioaficionados basado en terminal, para operadores que necesitan velocidad, fiabilidad y bajo consumo de recursos — en el shack, en una cumbre, en un field day o en una estación de club.

**Offline-first.** El registro local de QSO no requiere acceso a internet. Los datos de referencia, datos solares y prefijos DXCC almacenados en caché permanecen disponibles después de haber sido descargados una vez. Las integraciones de red como Wavelog, QRZ.com, DX Cluster y PSK Reporter requieren conectividad y se omiten en modo `--offline`.

**Construido para operación en campo.** CQOps está listo para QRP, es amigable con SOTA/POTA y funciona cómodamente en máquinas clase Raspberry Pi, portátiles antiguos y sistemas sin entorno de escritorio.

**Listo para estaciones de club.** CQOps soporta múltiples logbooks, perfiles de operador y presets de equipo. Cambie el logbook activo, el operador activo o el equipo activo con una sola tecla.

**Portátil por diseño.** CQOps es un solo binario escrito en Go. No tiene dependencia CGO ni servicios de sistema requeridos.

**Multiplataforma.** Windows, Linux y macOS son soportados en amd64 y arm64.

### Para Quién Es CQOps

- Operadores portátiles que necesitan registro rápido por teclado en hardware de bajo consumo.
- Activadores SOTA y POTA que registran offline y suben después.
- Estaciones de club con múltiples operadores compartiendo la misma estación.
- Equipos de field day usando máquinas compartidas o hardware clase Raspberry Pi.
- Operadores que prefieren un flujo de trabajo en terminal sobre una GUI de escritorio.

CQOps no pretende reemplazar libros de guardia de escritorio completos ni plataformas de logbook basadas en web. Se enfoca en registro rápido en terminal, operación en campo, uso offline y flujos de trabajo de estación compartida.

---

## Descarga e Instalación

> [Explorar todas las versiones →](https://github.com/szporwolik/cqops/releases)

### Windows

| Paquete | Enlace | Notas |
|---------|------|-------|
| **Instalador** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recomendado para la mayoría de usuarios. Añade CQOps al Menú Inicio y PATH. |
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

### Desde el Código Fuente

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
- Se recomienda un emulador de terminal moderno. En Windows, use Windows Terminal en lugar de la consola legacy.

### Opciones de Línea de Comandos

```bash
cqops              # Iniciar la TUI
cqops --offline    # Iniciar sin actividad de red
cqops --version    # Mostrar versión y salir
cqops --help       # Mostrar ayuda
```

---

## Primer Inicio — Asistente de Configuración

En el primer inicio, CQOps abre un asistente de configuración para los ajustes esenciales de la estación. Las integraciones de red se pueden omitir; el registro local funciona sin ellas.

1. **Estación y Logbook**  
   Configure el logbook inicial, indicativo de estación, operador y localizador de cuadrícula. Los campos opcionales incluyen referencias SOTA/POTA/WWFF, región IARU, zona CQ/ITU, DXCC y SIG/SIG Info. La configuración de Wavelog también está disponible aquí: URL, clave API, ID de perfil de estación, Update y Test.

2. **Equipo**  
   Configure un preset de equipo: nombre, modelo, antena, potencia y backend de radio. Los backends soportados son None, flrig y Hamlib rigctld. Los ajustes opcionales incluyen control de rotor Hamlib e integración UDP WSJT-X.

3. **Integraciones**  
   Configure la búsqueda de callbook QRZ.com: bandera de activación, nombre de usuario, contraseña enmascarada y Test.

4. **General**  
   Seleccione la zona horaria IANA. CQOps detecta la zona horaria del sistema por defecto y también proporciona una lista desplazable.

5. **Resumen**  
   Revise la configuración. Presione **Ctrl+S** para guardar e iniciar CQOps.

**Navegación del asistente:** **Ctrl+S** avanza después de la validación. **Esc** retrocede. **F10** sale. Espacio alterna casillas de verificación. Tab y Shift+Tab se mueven entre campos.

Todos los ajustes del asistente se pueden cambiar después desde el menú de configuración con **F9**.

---

## Inicio Rápido: Registrar su Primer QSO

1. **Instalar y ejecutar CQOps.**  
   Descargue el paquete para su plataforma, inicie `cqops` y complete el asistente de configuración con al menos su indicativo y localizador de cuadrícula.

2. **Usar el formulario QSO.**  
   El formulario QSO se abre en **F1**. Ingrese un indicativo; CQOps lo convierte automáticamente a mayúsculas. Si el equipo activo está conectado a través de flrig o Hamlib, la frecuencia, banda, modo y submodo se rellenan automáticamente. La fecha y hora se establecen en UTC actual.

3. **Moverse por los campos.**  
   Use **Tab**, **Shift+Tab** y **↑/↓**.

4. **Guardar el QSO.**  
   Presione **Enter** o **Ctrl+S**. Si aparece una advertencia **DUPE!**, presione **Enter** de nuevo para guardar de todos modos, o **Esc** para cancelar.

El nuevo QSO aparece inmediatamente en la tabla de QSO Recientes debajo del formulario.

---

## Vista General de la Pantalla Principal

```text
┌─ Barra de Estado ───────────────────────────────────────────────────────────────┐
│  CQOps v0.8.8  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Barra de Pestañas ──────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Área de Contenido Principal ────────────────────────────────────────────────────┤
│                                                                                  │
│  Formulario QSO, tabla, mapa, editor o contenido de pantalla activa              │
│                                                                                  │
├─ Barra de Ayuda ─────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Barra de Estado

La barra de estado muestra la versión de CQOps, el logbook activo, el equipo activo, el indicativo de estación y el operador activo. A la derecha se muestran las etiquetas de estado de integración y la hora en local (`L`) y UTC (`Z`).

**Colores de etiquetas:**

| Color | Significado |
|-------|-------------|
| Blanco/por defecto | Conectado o activo |
| Amarillo | Deshabilitado, conectando o fuera de línea esperado |
| Rojo | Error o desconectado |
| Acento + negrita | WSJT-X está transmitiendo |

Las etiquetas que pueden aparecer incluyen **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC** y **WL**.

### Barra de Pestañas

| Tecla | Pestaña | Pantalla |
|-----|-----|--------|
| F1 | QSO | Formulario QSO y tabla de QSO Recientes |
| F2 | QRZ | Vista de Partner: datos de callbook, mapa, estadísticas, foto |
| F4 | DXC | Spots de DX Cluster y filtros |
| F5 | HRD | Spots de PSK Reporter y mapa de propagación |
| F6 | REF | Búsqueda de referencias SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Navegador de planes de banda |
| F8 | LOG | Editor de Logbook, ADIF, sincronización Wavelog |
| F9 | CFG | Menús de configuración |

### Barra de Ayuda

La fila inferior muestra los atajos más relevantes para la pantalla activa. Presione **?** para la superposición completa de ayuda.

---

## Flujos de Trabajo Comunes

### Operación Portátil / SOTA / POTA

1. **Antes de salir de casa**, ejecute CQOps una vez con acceso a internet. Esto permite a CQOps poblar cachés como datos solares, datos REF y prefijos DXCC.
2. **Verifique la caché** antes de desconectarse. Compruebe que el panel Solar muestra datos y que la búsqueda REF en **F6** devuelve resultados.
3. **En el campo**, inicie CQOps con `cqops --offline`. La actividad de red se omite, lo que evita retrasos de servicios inalcanzables.
4. **Registre normalmente.** El registro local funciona sin internet.
5. **Suba después.** Cuando vuelva a estar en línea, abra el Editor de Logbook con **F8** y presione **w** para subir QSO no enviados a Wavelog.

### Estación de Club Compartida y Hot-Seat

1. **Añadir perfiles de operador:** abra **F9 → Operators**, luego presione **Ins** para cada operador. Ingrese su indicativo y nombre.
2. **Cambiar el operador activo:** en el formulario QSO, presione **Ctrl+O**. El operador activo se muestra en la barra de estado y se escribe en el campo `OPERATOR` de los QSO guardados.
3. **Usar registro hot-seat:** el operador A registra un QSO, el operador B presiona **Ctrl+O**, luego registra bajo su propio perfil de operador.
4. **Usar Retain cuando sea necesario:** active **Retain** si varios operadores necesitan registrar el mismo contacto sin reescribir el formulario completo.

Antes de guardar en una estación compartida, verifique el logbook activo y el operador activo en la barra de estado.

### Logbook Privado + de Club

Muchos operadores mantienen un logbook personal y uno o más logbooks de club.

1. **Crear logbooks:** abra **F9 → Logbooks**, luego presione **Ins** para cada logbook.
2. **Cambiar el logbook activo:** presione **Ctrl+L** en el formulario QSO. La barra de estado muestra el logbook activo.
3. **Mantener datos de estación separados:** cada logbook puede tener su propio indicativo de estación, configuración Wavelog, configuración de concurso y operadores.
4. **Registro dual rápido:** active **Retain**, guarde el QSO en un logbook, presione **Ctrl+L**, luego guárdelo de nuevo en el otro logbook si corresponde.

### Múltiples Equipos

1. **Crear presets de equipo:** abra **F9 → Rigs**, luego presione **Ins** para cada equipo.
2. **Establecer el backend:** use flrig o Hamlib para equipos controlados por CAT. Use None para equipos sintonizados manualmente.
3. **Cambiar el equipo activo:** presione **Ctrl+R** en el formulario QSO.
4. **Operar estaciones mixtas:** por ejemplo, use un equipo HF controlado por CAT y un equipo VHF/UHF manual en la misma sesión.
5. **Configurar WSJT-X por equipo:** cada preset de equipo puede tener su propia configuración UDP WSJT-X.

Cuando el equipo activo tiene control CAT, CQOps puede rellenar frecuencia, banda, modo y submodo automáticamente. Para equipos manuales, ingréselos usted mismo.

### FT8 / Auto-Registro WSJT-X

Cuando WSJT-X está conectado a través de UDP, CQOps puede registrar QSO digitales automáticamente desde mensajes ADIF de WSJT-X.

- Los QSO auto-registrados se guardan en el logbook activo.
- Los QSO auto-registrados duplicados se omiten.
- Los QSO auto-registrados heredan el ID de concurso activo.
- Los QSO aparecen inmediatamente en QSO Recientes.
- Si Wavelog está configurado y accesible, los QSO auto-registrados pueden subirse automáticamente.
- Si el operador de WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

Verifique el logbook activo, el operador activo y el concurso activo antes de sesiones digitales largas.

### Sincronización Wavelog

La sincronización Wavelog es opcional. CQOps siempre guarda los QSO localmente primero.

**Subida:** presione **w** en el Editor de Logbook (**F8**). CQOps sube QSO no enviados en lotes de 50 y rastrea el estado por QSO: no enviado, enviado o error.

**Descarga:** presione **Ctrl+W** en el Editor de Logbook. Las descargas son incrementales. CQOps obtiene QSO más nuevos que el `last_fetched_id` guardado para el logbook activo. Los duplicados se omiten.

Si una subida Wavelog falla, el QSO permanece en el logbook local y puede reintentarse más tarde. Purgar un logbook restablece el ID de obtención a `0`, permitiendo una descarga completa.

---

## Funciones Principales

### Registro de QSO

El formulario QSO (**F1**) es la pantalla principal de registro. Utiliza un diseño de tres columnas y puede auto-rellenar campos desde control de equipo, QRZ.com, búsqueda Wavelog, datos DXCC/prefijos y bases de datos REF.

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

⚠️ Los campos de intercambio aparecen solo cuando un concurso está activo. Los campos marcados con **(▼)** se ciclan con **PgUp/PgDn**.

La fila inferior contiene:

- **Comment** (Comentario)
- **Keep** — preserva el campo Comment entre QSOs; alternar con **Ctrl+T**
- **Retain** — preserva todo el formulario después de guardar

La línea de ruta/demora muestra distancia y azimut cuando ambos localizadores de cuadrícula son conocidos. También puede mostrar insignias como **DUPE!**, **New Call!** y **New DXCC!**.

### Fuentes de Auto-Relleno

| Fuente | Campos |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX si split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| Base de datos REF | Referencias SOTA, POTA, WWFF, IOTA |
| Búsqueda Wavelog | Estado worked/confirmed cuando está configurado |

### Registro de Concursos

Los concursos añaden campos de intercambio y manejo de seriales al formulario QSO.

Cree o configure un concurso en el Editor de Logbook (**F8**) con **Ins**. Establezca el nombre del concurso, fecha, ID de concurso ADIF y plantillas de intercambio.

Marcadores de plantilla soportados:

| Marcador | Reemplazado Por |
|--------|---------------|
| `@rst` | RST enviado o recibido |
| `@serial` | Número de serie auto-incrementable |
| `@call` | Su indicativo |
| `@grid` | Su localizador de cuadrícula |
| `@name` | Nombre del operador desde el perfil de operador |

Presione **Ctrl+C** para ciclar el concurso activo. Cuando un concurso está activo:

- el formulario QSO muestra campos de intercambio,
- los números de serie se auto-incrementan,
- QSO Recientes puede filtrar a QSO de concurso,
- la exportación ADIF preserva `CONTEST_ID`.

### Editor de Logbook

El Editor de Logbook (**F8**) se usa para gestión de QSO, importación/exportación ADIF, sincronización Wavelog y operaciones relacionadas con concursos.

**Edición en línea:** seleccione una fila con **↑/↓**, presione **Enter** o **e**, edite el QSO, luego guarde con **Ctrl+S**. Los cambios se reflejan en QSO Recientes inmediatamente.

### Importación y Exportación ADIF

CQOps soporta importación y exportación ADIF 3.1.7.

- **Ctrl+I** importa un archivo ADIF, valida registros, omite duplicados y muestra un resumen.
- **Ctrl+E** exporta QSOs. La exportación puede incluir todos los QSOs o QSOs filtrados por concurso.
- Los QSOs importados se marcan para subida Wavelog si la sincronización Wavelog está configurada.

### Favoritos

Los favoritos almacenan presets de frecuencia, modo y banda en 10 slots.

| Atajo | Acción |
|----------|--------|
| Alt+0–9 | Recuperar slot favorito |
| Alt+Shift+0–9 | Guardar frecuencia/modo/banda actual en slot |

Los favoritos se almacenan en la configuración y se comparten entre logbooks.

Ejemplo: para una configuración de llamada SOTA FM polaca, ingrese `145.55`, establezca modo `FM`, banda `2m`, luego presione **Alt+Shift+1**. Después, presione **Alt+1** para recuperarlo.

### Búsqueda REF

La pantalla REF (**F6**) busca referencias SOTA, POTA, WWFF e IOTA. Busque por prefijo, nombre o designador de referencia. Las referencias seleccionadas pueden rellenar el formulario QSO.

### Navegador de Planes de Banda

El Navegador de Planes de Banda (**F7**) proporciona acceso rápido a bandas de aficionados, rangos VHF/UHF, CB, PMR446 y presets de broadcast. Una frecuencia seleccionada puede usarse para sintonizar el equipo activo. Los datos de planes de banda también pueden exportarse como Markdown.

---

## Integraciones

### QRZ.com

La búsqueda QRZ.com requiere acceso a internet y una suscripción QRZ XML.

Presione **Ins** en el formulario QSO para rellenar campos de callbook como nombre, QTH, grid, país, zonas CQ/ITU, DXCC y continente. La vista Partner (**F2**) puede mostrar la foto del operador cuando está disponible.

### Wavelog

La integración Wavelog requiere acceso a internet. Soporta subida, descarga incremental y búsqueda worked/confirmed.

Wavelog se configura por logbook activo con URL, clave API e ID de perfil de estación. CQOps siempre guarda los QSOs localmente primero; el fallo de subida Wavelog no pierde datos.

Consulte [Sincronización Wavelog](#sincronización-wavelog).

### flrig

La integración flrig utiliza XML-RPC sobre HTTP. El endpoint predeterminado es `localhost:12345`.

CQOps puede leer frecuencia, modo y potencia desde flrig. La operación split se mapea como VFO A a Frequency y VFO B a Freq RX.

### Hamlib / rigctld

El control de equipo Hamlib utiliza el demonio TCP `rigctld`. CQOps puede consultar frecuencia, modo, VFO, split y potencia según el soporte del equipo.

Algunos equipos o backends Hamlib no soportan todas las consultas. CQOps maneja la falta de soporte de nombre VFO de forma segura cuando es posible.

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

La integración WSJT-X utiliza mensajes UDP de WSJT-X. CQOps analiza mensajes ADIF y puede auto-registrar QSOs completados.

La etiqueta del equipo se vuelve de color acento mientras WSJT-X transmite. Si el operador reportado por WSJT-X no coincide con el operador activo, CQOps muestra una advertencia.

Consulte [FT8 / Auto-Registro WSJT-X](#ft8--auto-registro-wsjt-x).

### DX Cluster

La integración DX Cluster utiliza una conexión telnet y requiere acceso a internet. El servidor predeterminado es `dxspots.com:7300`.

Los filtros incluyen banda, continente, modo y antigüedad/tiempo. Presione **Enter** en un spot para rellenar el formulario QSO, sintonizar el equipo activo y volver a la pantalla QSO. Presione **Space** para sintonizar sin rellenar el formulario. Presione **Backspace** para limpiar filtros.

### PSK Reporter

La integración PSK Reporter requiere acceso a internet. Proporciona spots de propagación, filtros de banda/tiempo/modo y un mapa mundial ASCII en **F5**.

### Datos Solares

Los datos solares incluyen SFI, número de manchas solares, índices A/K y condiciones banda por banda de hamqsl.com. Las actualizaciones en vivo requieren acceso a internet. Los datos en caché permanecen disponibles offline después de una obtención exitosa.

---

## Referencia de Configuración

La configuración de CQOps se almacena en:

| Plataforma | Ruta de configuración |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Las credenciales sensibles se almacenan por separado en `secrets.enc` en el mismo directorio de configuración. Los secretos están cifrados con una clave vinculada a la máquina, por lo que las credenciales deben reingresarse al mover una configuración a otra máquina.

Abra la configuración con **F9**.

| Menú | Configura |
|------|------------|
| Station | Indicativo, grid, zona CQ/ITU, región IARU, referencias |
| Rig | Presets de equipo, modelo, antena, potencia, backend, rotor, WSJT-X |
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

Los perfiles de operador contienen indicativo y nombre del operador. Presione **Ctrl+O** para ciclar el operador activo. El operador activo se guarda en el campo ADIF `OPERATOR` y se usa en las subidas Wavelog.

### Multi-Equipo

Los presets de equipo almacenan backend, modelo, antena, potencia, rotor y configuración WSJT-X. Presione **Ctrl+R** para ciclar el equipo activo.

### Secretos Cifrados

Desde v0.8.7, las credenciales se almacenan cifradas.

- **Archivo de secretos:** `secrets.enc`
- **Ubicación:** mismo directorio que `config.yaml`
- **Permisos Unix:** `0600` donde se soporte
- **Cifrado:** AES-256-GCM con clave vinculada a la máquina
- **Datos protegidos:** contraseña QRZ, login DX Cluster, claves API Wavelog
- **Migración:** secretos en texto plano de configuraciones antiguas migran en el primer inicio
- **Recuperación:** si `secrets.enc` está corrupto, CQOps inicia con una advertencia y solicita reingresar las credenciales

---

## Atajos de Teclado

### Globales

| Tecla | Acción |
|-----|--------|
| F1 | Formulario QSO y QSO Recientes |
| F2 | Vista Partner |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | Búsqueda REF |
| F7 | Navegador de Planes de Banda |
| F8 | Editor de Logbook |
| F9 | Configuración / menú principal |
| F10 | Salir |
| Ctrl+F9 | Visor de logs |
| ? | Superposición de ayuda |
| Ctrl+L | Ciclar logbook activo |
| Ctrl+R | Ciclar equipo activo |
| Ctrl+C | Ciclar concurso activo |
| Ctrl+O | Ciclar operador activo |
| Esc | Volver a pantalla anterior |

### Formulario QSO — F1

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

### Editor de Logbook — F8

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
| Enter | Rellenar formulario + sintonizar equipo + ir a QSO |
| Space | Sintonizar equipo al spot (permanecer en DXC) |
| Home | Filtro de banda adelante |
| End | Filtro de banda atrás |
| \\ | Filtro de continente |
| Ins | Filtro de modo adelante |
| Del | Filtro de modo atrás |
| PgUp | Filtro de tiempo adelante |
| PgDn | Filtro de tiempo atrás |
| Backspace | Limpiar todos los filtros |
| Esc / F4 | Volver al formulario QSO |

### Vista Partner — F2

| Tecla | Acción |
|-----|--------|
| F2 | Ciclo: Vista Partner → Foto → Volver |
| Esc / F1 | Volver al formulario QSO |

---

## Solución de Problemas

### La aplicación no inicia

- El terminal debe tener al menos 80×24 caracteres.
- En Windows, use Windows Terminal, no la consola legacy `cmd.exe`.
- Pruebe `cqops --offline` para descartar problemas de red.
- Revise los logs: `~/.local/share/cqops/logs/` (Linux), `~/Library/Application Support/cqops/logs/` (macOS) o `%APPDATA%\cqops\logs\` (Windows).

### El equipo no se conecta

- **flrig:** verifique que flrig esté funcionando y el puerto coincida (predeterminado `12345`).
- **Hamlib:** verifique que rigctld esté funcionando y el puerto TCP sea correcto.
- Color de etiqueta de estado: blanco = conectado, amarillo = conectando/deshabilitado, rojo = error.
- Los toasts de reconexión suprimidos son normales — CQOps reintenta en segundo plano.

### WSJT-X no auto-registra

- Verifique la configuración UDP de WSJT-X: Settings → Reporting → UDP Server.
- WSJT-X debe ser versión 2.6 o superior.
- La etiqueta de estado debería ser blanca (por defecto) cuando WSJT-X está funcionando.

### La subida Wavelog falla

- Verifique URL, clave API e ID de perfil de estación en la configuración.
- Etiqueta de estado: blanco = accesible, amarillo = deshabilitado/sin internet, rojo = error.
- Los errores de subida se muestran como toasts; los QSOs permanecen guardados localmente.
- Los fallos individuales de QSO no bloquean el resto del lote.

### Problemas con el archivo de configuración

- Configuración: `~/.config/cqops/config.yaml` (Linux/macOS) o `%APPDATA%\cqops\config.yaml` (Windows).
- Secretos: `secrets.enc` en el mismo directorio.
- Si la configuración está corrupta, elimínela y reinicie — el asistente creará una nueva.
- El campo `last_fetched_id` solo aparece después de una descarga Wavelog exitosa.

### Rendimiento

- Desactive el renderizado de mapa y el panel solar en ajustes General.
- Cierre pestañas no utilizadas (DXC, PSK).
- Ejecute con `--offline` si la red no es confiable.

### Reportar Errores

Active el **modo Debug** antes de reproducir un problema — F9 → General → Debug, o establezca `debug: true` en la configuración. Los logs completos se escriben en el directorio de logs específico de la plataforma.

Reporte problemas en [GitHub Issues](https://github.com/szporwolik/cqops/issues) con:
- Versión de CQOps (`cqops --version`)
- Sistema operativo y emulador de terminal
- Pasos para reproducir
- Log de debug
