---
title: Manual de usuario de CQOps
description: Guía práctica para configurar CQOps y registrar contactos en la estación o en el campo
---

# Manual de usuario de CQOps

CQOps es un programa de registro de radioaficionado manejado con el teclado, para estación fija, portable, radioclub y concursos ocasionales. Los QSO se guardan primero en el ordenador; los servicios de Internet son opcionales. Empiece con el registro manual y añada control del equipo o servicios en línea cuando los necesite.

Los nombres de menús y campos corresponden a la interfaz inglesa. Los atajos dependen de la pantalla activa: consúltelos en la barra inferior o con **?**.

## Contenido

1. [Instalación](#installation)
2. [Configuración inicial](#setup)
3. [Primer QSO](#first-qso)
4. [Pantallas y estado](#screens)
5. [Registro diario](#logging)
6. [Perfiles de estación](#profiles)
7. [Libro de guardia y copias de seguridad](#logbook)
8. [Equipo y modos digitales](#radio)
9. [Servicios en línea](#online)
10. [GPS y APRS](#position)
11. [Operación portable](#portable)
12. [Concursos](#contests)
13. [CQOps Live](#dashboard)
14. [Atajos de teclado](#keys)
15. [Solución de problemas y ayuda](#help)

<a id="installation"></a>

## Instalación

Descargue CQOps de la [página de versiones](https://github.com/szporwolik/cqops/releases). El terminal necesita al menos 75 × 24 caracteres; 80 × 43 o más resulta más cómodo.

| Sistema | Instalación |
|---|---|
| Windows | Descargue `cqops-setup.exe` o extraiga `cqops-windows-portable.zip` para usarlo sin instalar. Se recomienda Windows Terminal. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Descargue el `.deb` adecuado: `amd64` para la mayoría de PC Intel/AMD, `arm64` para ARM de 64 bits o `armhf` para Raspberry Pi OS de 32 bits. Ábralo con el instalador de paquetes. |
| Fedora, RHEL, Rocky, AlmaLinux | Use los comandos del repositorio indicados abajo. |
| Arch, Manjaro, CachyOS | Instale el paquete AUR con `paru -S cqops-bin` o `yay -S cqops-bin`. |
| Otros sistemas Linux | Descargue y extraiga el archivo Linux `.tar.gz` correspondiente al procesador. |
| macOS | Descargue `cqops-darwin-arm64` para Apple Silicon o `cqops-darwin-amd64` para Intel. Use los comandos siguientes. |

En sistemas basados en Debian también puede instalar desde el repositorio:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

En sistemas basados en Fedora:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

En macOS, ejecute lo siguiente en la carpeta de descarga, sustituyendo `FILE` por el nombre exacto del archivo:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Inicie `cqops` o el programa portable extraído. `cqops --offline` inicia sin conexión, `cqops --version` muestra la versión y `cqops --help` las opciones de inicio. Exporte los registros antes de actualizar.

<a id="setup"></a>

## Configuración inicial

El asistente pide nombre del libro, indicativo de estación, locator Maidenhead y continente. El **indicativo de estación** es el usado al transmitir; el **perfil de operador** identifica a la persona que opera.

**Ctrl+A** muestra referencias de estación y zonas CQ/ITU opcionales. Su referencia SOTA/POTA/WWFF pertenece a los ajustes de estación/libro; las referencias del formulario QSO corresponden al contacto. Configure después la región IARU en **F9 → Logbooks**.

Cree un perfil de equipo con nombre, antena y potencia. Elija **None** para introducir frecuencia y modo manualmente, **flrig** o **Hamlib**. Configure las conexiones opcionales después de comprobar el registro básico.

**Tab / Shift+Tab** cambia de campo, **Space** modifica opciones y **Save & Next** continúa. **Esc** retrocede; **F10** sale. Revise el resumen y guarde. CQOps detecta la zona horaria del ordenador; fecha y hora de los QSO se registran en UTC. Compruebe el reloj antes de operar.

<a id="first-qso"></a>

## Primer QSO

1. Pulse **F1** y compruebe libro, indicativo, operador, equipo y concurso activos.
2. Introduzca el indicativo del corresponsal. **Ins** realiza una consulta si está configurada. La prioridad indica confianza en los datos: QRZ.com (100) > HamQTH (90) > Callook.info (80) > QRZ.RU (70) > cuaderno local (60) > Wavelog (10), con CTY.DAT siempre al final; los de menor prioridad solo rellenan campos vacíos.
3. Compruebe fecha/hora UTC, frecuencia en MHz, banda, modo e informes enviados/recibidos.
4. Añada nombre, QTH, locator, referencia o comentario si procede.
5. Pulse **Enter**. El contacto aparece en Recent QSOs.

Si **DUPE!** solicita confirmación, otro **Enter** guarda igualmente y **Esc** cancela la confirmación. La advertencia invita a revisar el contacto; no demuestra que deba descartarse.

<a id="screens"></a>

## Pantallas y estado

| Tecla | Pantalla | Función |
|---|---|---|
| F1 | QSO | Introducir contactos y ver QSO recientes |
| F2 | Partner | Datos del corresponsal, mapa, estadísticas, foto |
| F3 | APRS | Estaciones cercanas |
| F4 | DX Cluster | Spots y filtros |
| F5 | PSK Reporter | Informes de recepción digital |
| F6 | References | Búsqueda SOTA, POTA, WWFF, IOTA |
| F7 | Band Plan | Frecuencias y ajustes predefinidos |
| F8 | Logbook | Editar, importar, exportar y sincronizar |
| F9 | Configuration | Ajustes de estación y servicios |
| F10 | Quit | Salir de CQOps |

La barra superior muestra la configuración activa, hora local (**L**) y UTC (**Z**). Blanco suele significar activo; amarillo, desactivado/conectando/en espera; rojo, error. WSJT se destaca al transmitir. **WL!** indica una clave Wavelog antigua no admitida.

<a id="logging"></a>

## Registro diario

Use **Tab / Shift+Tab** entre campos y **PgUp / PgDn** para cambiar banda, modo o submodo. **Shift+Backspace** borra el campo; **Del** borra el formulario. Compruebe **Freq RX** en operación split.

**Keep** conserva el comentario al guardar. **Retain** conserva todo el formulario: revise indicativo, hora, informes y referencias antes del siguiente contacto. Los campos de intercambio solo aparecen con un concurso activo. Use **SIG / SIG Info** para otros datos de grupos de interés especial.

Con ambos locators conocidos, CQOps muestra distancia y rumbo. El callbook puede indicar el domicilio en vez del lugar portable actual; verifíquelo. Los indicadores de nuevo indicativo, nuevo DXCC y duplicado ayudan a valorar el contacto.

**F6** busca referencias por nombre o identificador y puede rellenar la del corresponsal. **F7** muestra planes de banda y puede sintonizar un equipo conectado. Son ayudas operativas, no autorizaciones para transmitir: compruebe sus atribuciones y el plan local.

Tres favoritos compartidos guardan frecuencia, modo y banda:

| Posición | Recuperar | Guardar valores actuales |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Perfiles de estación

Cree libros, operadores, equipos y concursos en sus menús **F9**; **Ins** añade una entrada. Desde QSO:

| Atajo | Cambiar |
|---|---|
| Ctrl+L | Libro |
| Ctrl+O | Operador |
| Ctrl+R | Equipo |
| Ctrl+C | Concurso |

Cada libro mantiene datos de estación y ajustes Wavelog/APRS separados. El perfil de operador identifica a la persona; el indicativo se guarda en ADIF `OPERATOR`. Los perfiles de equipo incluyen material, potencia, control de radio/rotor y WSJT-X. Compruebe el estado después de cambiar, especialmente con registro digital automático.

Otros menús **F9** cubren pantalla, unidades, zona horaria, callbooks, integraciones y avisos sonoros.

<a id="logbook"></a>

## Libro de guardia y copias de seguridad

En **F8**, seleccione un QSO y pulse **Enter** o **e** para editar. Guarde con **Enter** y confirme. **Delete** elimina el contacto elegido. Haga una copia antes de cambios masivos; **Ctrl+P** elimina todos los QSO, no realiza una búsqueda.

| Atajo F8 | Acción |
|---|---|
| Ctrl+I | Importar ADIF, validar registros y omitir duplicados |
| Ctrl+E | Exportar todos los contactos o una selección por concurso |
| Ctrl+W | Subir contactos pendientes a Wavelog |
| Alt+W | Descargar desde Wavelog |

Revise el resumen de importación y la selección de exportación. Los contactos importados pueden subirse después a Wavelog. CQOps admite ADIF 3.1.7 y conserva identificadores de concurso e intercambios. Guarde copias separadas de cada libro, preferiblemente en otro dispositivo. ADIF respalda los contactos, no todos los ajustes ni credenciales.

La configuración está en `~/.config/cqops/config.yaml` en Linux/macOS y `%APPDATA%\cqops\config.yaml` en Windows. Las credenciales se guardan aparte en `secrets.enc`; introdúzcalas de nuevo al cambiar de ordenador. No borre la configuración como primer paso de diagnóstico.

<a id="radio"></a>

## Equipo y modos digitales

### Control del equipo

En **F9 → Rigs**, seleccione flrig o Hamlib y haga coincidir la conexión. Inicie antes flrig o `rigctld`. flrig normalmente usa `localhost:12345`. Las lecturas de frecuencia, modo, split y potencia dependen del equipo. Con **None**, introdúzcalas manualmente.

### WSJT-X

Use WSJT-X 2.6 o posterior. Haga coincidir **Settings → Reporting → UDP Server** con los ajustes UDP del perfil CQOps activo. Registre un QSO de prueba completado en WSJT-X y confirme que aparece en CQOps.

Los QSO recibidos usan libro y concurso activos; los duplicados se omiten. Compruebe operador e indicador WSJT antes de empezar. CQOps advierte de discrepancias de operador. Puede seguir la subida a Wavelog si está configurada. Elija Mode/Submode correctamente: FT8 se exporta como FT8, y FT4/FT2 como MFSK con su submodo.

### Control del rotor

El control Hamlib `rotctld` es experimental. Verifique sentido y límites físicos. Mantenga una forma segura de detenerlo: una configuración incorrecta puede dañar antena, rotor o línea de alimentación.

| Atajo | Acción |
|---|---|
| Alt+, / Alt+. | Acimut −5° / +5° |
| Alt+' / Alt+; | Elevación −5° / +5° |
| Alt+\ | Orientar al rumbo calculado |
| Alt+/ | Detener movimiento |

<a id="online"></a>

## Servicios en línea

### Callbooks

Configure proveedores y prioridad en **F9 → Callbook**, después pulse **Ins** en QSO. CQOps consulta los habilitados por orden. La búsqueda del indicativo base puede omitir prefijos/sufijos portables; compruebe la ubicación devuelta.

| Proveedor | Acceso |
|---|---|
| QRZ.com | Suscripción XML y credenciales |
| HamQTH | Cuenta gratuita |
| QRZ.RU | Acceso API separado del acceso a la web |
| Callook.info | Indicativos de EE. UU.; sin cuenta |

**F2** muestra al corresponsal. Las fotos dependen del proveedor y terminal; **Kitty Graphics**, experimental en General, requiere Kitty, Ghostty, WezTerm u otro terminal compatible.

### Wavelog

Configure URL, token API v2 (`wl2_…`) y perfil de estación por libro. No se admiten claves v1. Seleccionar una estación Wavelog puede rellenar datos locales: revise indicativo, locator y referencias antes de guardar.

Los QSO se guardan primero localmente. Reintente subidas fallidas con **F8 → Ctrl+W**; **Alt+W** descarga contactos. Al abrir un QSO vinculado para editar, CQOps puede actualizarlo desde Wavelog. Las ediciones y eliminaciones en línea afectan también a la copia remota. Lea la confirmación, especialmente sin conexión; no suponga que un cambio solo local haya llegado a Wavelog.


Estaciones de club: use la clave `wl2_` del propietario junto con la opción **Estación de club compartida** del formulario del cuaderno. Los contactos sincronizados pasan a ser de solo lectura: las ediciones y eliminaciones se hacen en Wavelog y CQOps nunca envía PATCH ni DELETE para ellos. Los contactos nuevos se suben con normalidad y se atribuyen al operador activo (o al distintivo de la estación si no hay operador). La clave API se guarda cifrada y, con esta opción activada, nunca vuelve a mostrarse una vez guardada — deje el campo vacío para conservarla o escriba una nueva para reemplazarla.
### DX Cluster y propagación

Configure DX Cluster en Integrations y abra **F4**. **b / c / m / t** filtran banda, continente del anunciante, modo y antigüedad. **Backspace** borra filtros. **Enter** rellena QSO, sintoniza el equipo conectado y vuelve a F1; **Space** sintoniza sin salir del cluster.

En F1, **Ctrl+S** abre el diálogo de spot y **Ctrl+P** toma el indicativo del spot mostrado más cercano. Verifíquelo antes de enviar. **F5** muestra informes PSK Reporter, no garantiza propagación actual. Solar muestra condiciones HamQSL; los valores almacenados pueden ser antiguos. **F5 está desactivado por defecto: active PSK Reporter en Integraciones.**

<a id="position"></a>

## GPS y APRS

### GPS

Configure GPS serie o GPSD en Integrations. Active **Grid from GPS** en estación/libro para usar su locator en contactos, rumbos, APRS y panel. GPS rojo significa error; amarillo, sin posición; blanco, posición obtenida. Verifique el locator antes de operar. Elija 6, 8 o 10 caracteres; más caracteres no garantizan mayor precisión del receptor.

### APRS

| Servicio | Conexión |
|---|---|
| APRS-IS | Servidor APRS de Internet |
| KISS | TNC físico serie y radio |
| KISS Server | TNC TCP como Dire Wolf; puede ser local |

Seleccione el servicio en **F9 → Integrations → APRS**. Ajuste indicativo/SSID, símbolo, comentario, alcance e intervalo en **F9 → Logbooks → [active logbook] → APRS**. Active **APRS TX** y **Send beacons** solo si desea transmitir. La recepción sola muestra **APRS-RX**. Las balizas revelan su posición: compruébela y tenga en cuenta quién podrá recibirla.

Los intervalos automáticos son de al menos cinco minutos. **F3** muestra estaciones oídas recientemente: flechas seleccionan, **Enter** rellena QSO, **d / t / s** filtran distancia/antigüedad/tipo, **Backspace** limpia y **b** envía inmediatamente una baliza configurada. Las balizas GPS requieren **Grid from GPS** y una posición válida.

<a id="portable"></a>

## Operación portable

Antes de salir, seleccione el libro portable y compruebe indicativo, locator, referencia de activación, equipo, antena y potencia. Pruebe toda la estación y ejecute CQOps en línea para actualizar referencias y prefijos. Verifique que **F6** encuentra las referencias necesarias. Exporte una copia.

El registro local funciona sin Internet. `cqops --offline` omite funciones de red; no dependa de consultas en directo ni sincronización. Pruebe el equipo de red local en el modo elegido antes de salir. Los datos en caché pueden estar desactualizados.

Al volver, revise cantidad de QSO y referencias, exporte ADIF, guarde una copia y suba contactos pendientes a Wavelog si lo usa. Compruebe el formato exigido por cada programa de diplomas y convierta el archivo si hace falta.

<a id="contests"></a>

## Concursos

CQOps permite concursos ocasionales, intercambios, números correlativos y tasas de QSO. No es un sistema completo de puntuación o envío de registros. Para operación avanzada, use un programa dedicado.

En **F9 → Contests**, pulse **Ins** y defina nombre, fecha, identificador ADIF, número inicial y plantillas de intercambio enviado/recibido.

| Marcador | Valor |
|---|---|
| `@rst` | Informe enviado o recibido |
| `@serial` | Número correlativo |
| `@cqz` / `@mycqz` | Zona CQ del contacto / propia |
| `@itu` / `@myitu` | Zona ITU del contacto / propia |
| `@grid` / `@mygrid` | Locator del contacto / propio |

En **F1**, **Ctrl+C** cambia de concurso. Compruebe intercambio y próximo número antes de transmitir. La barra muestra total, próximo número y tiempos; ventanas anchas añaden estadísticas de ritmo. Al terminar, vuelva a operar sin concurso activo.

Para exportar, abra **F8**, elija el filtro de concurso con **Ctrl+C**, luego **Ctrl+E** y compruebe la selección. La salida es ADIF, no Cabrillo. Siga las normas de formato y entrega del organizador.

<a id="dashboard"></a>

## CQOps Live

Active **F9 → Integrations → HTTP Server** y guarde con **Ctrl+S**. En el ordenador CQOps abra `http://localhost:8073`.

La dirección predeterminada `0.0.0.0` permite acceso local sujeto al cortafuegos. En otro dispositivo use la IP del ordenador CQOps y el puerto `8073`. `127.0.0.1` limita el acceso al propio ordenador. Use una red de confianza; no redirija el puerto a Internet.

El panel actualiza contacto actual, mapas QSO, contactos recientes, ritmos, operadores, APRS y datos disponibles de propagación/meteorología. Las capas de Internet pueden faltar sin conexión. Header 1, Header 2, Logo URL y Event Start personalizan el evento; la fecha inicial filtra estadísticas y listas de QSO.

<a id="keys"></a>

## Atajos de teclado

| Pantalla | Teclas | Acción |
|---|---|---|
| General | ? / Esc / F10 | Ayuda / atrás / salir |
| QSO | Tab / Shift+Tab | Campo siguiente / anterior |
| QSO | Enter / Ins | Guardar / consultar |
| QSO | Shift+Backspace / Del | Borrar campo / formulario |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Cambiar libro / operador / equipo / concurso |
| Libro | ↑ / ↓, PgUp / PgDn, Home / End | Selección, página, primera / última fila |
| Libro | Enter o e / Delete | Editar / eliminar QSO seleccionado |
| Libro | Ctrl+I / Ctrl+E | Importar / exportar ADIF |
| Libro | Ctrl+W / Alt+W | Subir / descargar Wavelog |
| Libro | Ctrl+C / Backspace | Filtro de concurso / borrar búsqueda |

Los atajos dependen de la pantalla: **Ctrl+C no sale de CQOps**. En portátiles, las teclas de función pueden requerir **Fn**. Si el terminal captura un atajo, revise su teclado y la barra de ayuda CQOps.

<a id="help"></a>

## Solución de problemas y ayuda

| Problema | Comprobar primero |
|---|---|
| Inicio o pantalla incompleta | Tamaño del terminal, Windows Terminal en Windows, probar `cqops --offline` |
| Equipo desconectado | Perfil activo, flrig/rigctld iniciado, modelo, puerto serie, velocidad, host/puerto, otro programa ocupando el puerto |
| Falta QSO de WSJT-X | UDP coincidente, indicador WSJT, QSO realmente registrado en WSJT-X, libro activo |
| Error Wavelog | URL, token `wl2_`, perfil, Internet; los QSO locales se conservan |
| Sin posición GPS | Puerto/velocidad o dirección GPSD, vista del cielo, posición válida, Grid from GPS |
| APRS no envía balizas | Libro, APRS TX y Send beacons, indicativo/SSID, TNC/radio o Internet |
| Panel inaccesible | Servidor activo, IP/puerto correctos, cortafuegos; localhost se refiere al dispositivo del navegador |

Use **F9** antes de modificar archivos de configuración. Reintroduzca credenciales si CQOps informa de un problema con los secretos o si cambió de ordenador.

Si hace falta, active **F9 → General → Debug**, reproduzca el problema con seguridad y recoja el registro de diagnóstico; desactive después la depuración.

| Sistema | Registros de diagnóstico |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Notifique problemas en [GitHub Issues](https://github.com/szporwolik/cqops/issues). Incluya versión CQOps, sistema, terminal, pasos y registro relevante. Elimine contraseñas, tokens API e información privada antes de compartir.
