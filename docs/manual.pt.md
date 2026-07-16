---
title: Manual do Usuário do CQOps
description: Guia do usuário para instalar, configurar e usar o CQOps — um logger de radioamador rápido e orientado ao terminal
---

# Manual do Usuário do CQOps

O CQOps é um logger de radioamador rápido e orientado ao terminal, destinado a operadores que desejam registrar contatos com confiabilidade pelo teclado e com baixo consumo de recursos do sistema. Ele foi projetado para uso na estação, operações portáteis, estações de clube, field days e computadores como dispositivos da classe Raspberry Pi ou notebooks mais antigos.

O CQOps sempre salva os QSOs localmente primeiro. As integrações baseadas na internet são opcionais.

<a id="contents"></a>
## Conteúdo

1. [O que é o CQOps](#what-cqops-is)
2. [Download e instalação](#download-and-installation)
3. [Primeira execução](#first-launch)
4. [Registre seu primeiro QSO](#log-your-first-qso)
5. [Tela principal](#main-screen)
6. [Fluxos de trabalho comuns](#common-workflows)
7. [Registro de QSOs](#qso-logging)
8. [Editor do logbook e ADIF](#logbook-editor-and-adif)
9. [Contests](#contests)
    - [Configurando um contest](#setting-up-a-contest)
    - [Barra de status inferior](#bottom-status-bar)
    - [Painel de estatísticas do contest](#contest-statistics-panel)
    - [Exportação ADIF do contest](#contest-adif-export)
    - [Comportamento do modo contest](#contest-mode-behavior)
10. [Favoritos, referências e planos de banda](#favorites-references-and-band-plans)
11. [Integrações](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuração](#configuration)
14. [Atalhos de teclado](#keyboard-shortcuts)
15. [Solução de problemas](#troubleshooting)
16. [Relatando bugs](#reporting-bugs)

---

<a id="what-cqops-is"></a>
## O que é o CQOps

O CQOps foi desenvolvido em torno de entrada rápida de QSOs, registro local em primeiro lugar e operação prática em campo.

<a id="main-ideas"></a>
### Ideias principais

- **Operação orientada ao terminal** — otimizada para uso pelo teclado.
- **Registro offline-first** — o registro local de QSOs funciona sem acesso à internet. Inclui um mapa-múndi incorporado para o dashboard que funciona totalmente offline.
- **Baixo consumo de recursos** — adequado para sistemas da classe Raspberry Pi, notebooks antigos e PCs compartilhados em estações.
- **Design portátil** — distribuído como um único binário em Go.
- **Vários logbooks** — útil para logs pessoais, portáteis, de contests e de clubes.
- **Vários operadores** — útil para fluxos de trabalho hot-seat e estações de clube compartilhadas.
- **Vários rádios** — cada preset de rádio pode manter suas próprias configurações de backend e WSJT-X.
- **Integrações opcionais** — callbook com vários provedores (QRZ.com, HamQTH, QRZ.RU, Callook.info), Wavelog, DX Cluster, PSK Reporter, GPS, APRS, controle do rádio, controle do rotor, dados solares e o dashboard CQOps Live no navegador.

O registro local não exige acesso à internet. Os recursos de rede são ignorados no modo `--offline`.

<a id="who-cqops-is-for"></a>
### Para quem é o CQOps

O CQOps é uma boa opção para:

- operadores portáteis,
- ativadores SOTA e POTA,
- estações de clube,
- equipes de field day,
- operadores que preferem um fluxo de trabalho no terminal,
- estações que precisam alternar rapidamente entre operadores, logbooks ou rádios.

O CQOps não pretende substituir todos os recursos de um logger completo para desktop ou de uma plataforma de logbook na web. Seu foco é o registro rápido no terminal, a operação em campo, o uso offline e os fluxos de trabalho de estações compartilhadas.

<a id="club-and-shared-station-use"></a>
### Uso em clubes e estações compartilhadas

O CQOps foi criado pensando em ambientes de radioclubes. O operador ativo está sempre visível na barra de status — **um único olhar** informa quem está conectado no momento. A troca de operadores exige apenas uma tecla (`Ctrl+O`) e entra em vigor imediatamente, com o indicativo e o nome do operador gravados em cada QSO subsequente. Sem logout, sem solicitação de senha e sem interrupção.

Logbooks, presets de rádio e contests são alternados da mesma forma — `Ctrl+L`, `Ctrl+R`, `Ctrl+C`. Uma estação de clube com operadores em rodízio, vários rádios e diversos contests ativos pode mudar de contexto em menos de um segundo, sem tocar no mouse.

Para field days e eventos públicos, o **CQOps Live dashboard** projeta em uma tela grande um mapa em tempo real, o fluxo de QSOs e estatísticas — visitantes e membros do clube podem acompanhar a operação sem se aglomerar ao redor do terminal do operador. Basta habilitar a integração HTTP Server e acessar com qualquer dispositivo que tenha um navegador da web.

---

<a id="download-and-installation"></a>
## Download e instalação

Veja todas as versões:

<https://github.com/szporwolik/cqops/releases>

<a id="windows"></a>
### Windows

| Pacote | Link | Observações |
|---|---|---|
| Instalador | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recomendado para a maioria dos usuários. Adiciona o CQOps ao menu Start e ao PATH. |
| ZIP portátil | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extraia e execute sem instalar. |


<a id="linux--debian--ubuntu"></a>
### Linux — Debian / Ubuntu

| Arquitetura | Link | Use em |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Maioria dos PCs Intel/AMD |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | Sistemas ARM de 64 bits |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | Raspberry Pi OS de 32 bits |

Instale o pacote baixado:

```bash
sudo dpkg -i cqops_*.deb
```

<a id="linux--portable-tarball"></a>
### Linux — arquivo Tarball portátil

| Arquitetura | Link | Use em |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Maioria dos PCs Intel/AMD |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | Sistemas ARM de 64 bits |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | Raspberry Pi OS de 32 bits |

<a id="macos"></a>
### macOS

| Arquitetura | Link | Use em |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | Macs M1/M2/M3 |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Macs Intel |

Instale manualmente:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

<a id="build-from-source"></a>
### Compilar a partir do código-fonte

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Compilações a partir do código-fonte exigem Go 1.26 ou mais recente.

<a id="terminal-requirements"></a>
### Requisitos do terminal

| Requisito | Valor |
|---|---|
| Tamanho mínimo do terminal | 80×24 caracteres |
| Tamanho recomendado do terminal | 80×43 caracteres ou maior |
| Terminal recomendado no Windows | Windows Terminal |
| Terminal com gráficos Kitty | [Kitty](https://sw.kovidgoyal.net/kitty/), [Ghostty](https://ghostty.org/) ou [WezTerm](https://wezfurlong.org/wezterm/) |

<a id="basic-commands"></a>
### Comandos básicos

```bash
cqops              # Inicia a TUI
cqops --offline    # Inicia sem atividade de rede
cqops --version    # Exibe a versão e encerra
cqops --help       # Exibe a ajuda
```

---

<a id="first-launch"></a>
## Primeira execução

Na primeira execução, o CQOps abre o assistente de configuração. Para o registro local, somente as informações essenciais da estação são obrigatórias. As integrações de rede podem ser ignoradas e configuradas posteriormente.

<a id="wizard-pages"></a>
### Páginas do assistente

| Página | O que configura |
|---|---|
| Station & Logbook | Logbook inicial, indicativo da estação, operador, grid locator, referências e zonas opcionais, URL/API/ID do station profile do Wavelog |
| Rig | Preset do rádio, modelo, antena, potência, backend, rotor opcional e configurações UDP opcionais do WSJT-X |
| Integrations | Configurações de consulta ao callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info) |
| General | Fuso horário IANA |
| Summary | Revisar e salvar |

Os backends de rádio compatíveis são:

- None,
- flrig,
- Hamlib `rigctld`.

<a id="wizard-navigation"></a>
### Navegação no assistente

| Tecla | Ação |
|---|---|
| Ctrl+S | Validar e continuar; em Summary, salvar e iniciar o CQOps |
| Esc | Voltar |
| F10 | Sair |
| Tab / Shift+Tab | Mover entre os campos |
| Space | Alternar caixas de seleção |

Você pode alterar posteriormente as configurações do assistente com **F9**.

---

<a id="log-your-first-qso"></a>
## Registre seu primeiro QSO

1. Inicie o CQOps:

   ```bash
   cqops
   ```

2. Conclua o assistente de configuração informando pelo menos seu indicativo e grid locator.

3. Abra o formulário de QSO com **F1**.

4. Digite o indicativo do contato. O CQOps converte automaticamente os indicativos para letras maiúsculas.

5. Preencha os demais campos. Se o rádio ativo estiver conectado por flrig ou Hamlib, o CQOps poderá preencher automaticamente frequência, banda, modo e submodo.

6. Pressione **Enter** para salvar.

7. Se aparecer um aviso **DUPE!**, pressione **Enter** novamente para salvar mesmo assim ou **Esc** para cancelar.

O QSO salvo aparece imediatamente na tabela Recent QSOs.

---

<a id="main-screen"></a>
## Tela principal

O CQOps usa um layout fixo no terminal:

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

<a id="status-bar"></a>
### Barra de status

A barra de status mostra:

- versão do CQOps,
- logbook ativo,
- rádio ativo,
- indicativo da estação,
- operador ativo,
- rótulos de status das integrações,
- hora local marcada com `L`,
- hora UTC marcada com `Z`.

Os rótulos comuns incluem **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL** e **GPS**. O rótulo GPS segue a mesma convenção de cores — vermelho quando desconectado, amarelo quando conectado mas sem fix e branco quando uma posição foi obtida.

| Cor | Significado |
|---|---|
| Branco/padrão | Conectado ou ativo |
| Amarelo | Desabilitado, conectando ou offline conforme esperado |
| Vermelho | Erro ou desconectado |
| Cor de destaque + negrito | O WSJT-X está transmitindo |

<a id="main-tabs"></a>
### Abas principais

| Tecla | Aba | Tela |
|---|---|---|
| F1 | QSO | Formulário de QSO e Recent QSOs |
| F2 | QRZ | Partner view: dados do callbook, mapa, estatísticas, foto |
| F4 | DXC | Spots e filtros do DX Cluster |
| F5 | HRD | Spots do PSK Reporter e mapa de propagação |
| F6 | REF | Busca de referências SOTA/POTA/WWFF/IOTA |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook Editor, ADIF, sincronização com Wavelog |
| F9 | CFG | Menus de configuração |

A barra de ajuda mostra os atalhos relevantes para a tela ativa. Pressione **?** para abrir a sobreposição de ajuda completa.

---
<a id="common-workflows"></a>
## Fluxos de trabalho comuns

<a id="portable-sota-or-pota-operation"></a>
### Operação portátil, SOTA ou POTA

Antes de sair de casa:

1. Execute o CQOps uma vez com acesso à internet.
2. Deixe o CQOps baixar ou atualizar dados em cache, como dados solares, dados REF e prefixos DXCC.
3. Verifique se o painel Solar mostra dados.
4. Verifique se a busca REF em **F6** retorna resultados.

Em campo:

1. Inicie o CQOps no modo offline:

   ```bash
   cqops --offline
   ```

2. Registre os contatos normalmente. Os QSOs são salvos localmente.
3. Quando estiver novamente online, abra **F8** e pressione **w** para enviar ao Wavelog os QSOs ainda não enviados.

<a id="shared-club-station-and-hot-seat-logging"></a>
### Estação de clube compartilhada e registro hot-seat

1. Abra **F9 → Operators**.
2. Pressione **Ins** para adicionar perfis de operador.
3. No formulário de QSO, pressione **Ctrl+O** para alternar o operador ativo.
4. Confira o operador ativo na barra de status antes de salvar.
5. Use **Retain** quando vários operadores precisarem registrar contatos semelhantes sem redigitar todo o formulário.

O operador ativo é salvo no campo ADIF `OPERATOR`.

<a id="personal-and-club-logbooks"></a>
### Logbooks pessoais e de clube

1. Abra **F9 → Logbooks**.
2. Pressione **Ins** para criar cada logbook.
3. No formulário de QSO, pressione **Ctrl+L** para alternar o logbook ativo.
4. Confira o logbook ativo na barra de status antes de salvar.

Cada logbook pode manter seus próprios dados da estação, configurações do Wavelog, configurações de contest e operadores.

<a id="multiple-rigs"></a>
### Vários rádios

1. Abra **F9 → Rigs**.
2. Pressione **Ins** para criar presets de rádio.
3. Selecione o backend: None, flrig ou Hamlib.
4. No formulário de QSO, pressione **Ctrl+R** para alternar o rádio ativo.

Um preset de rádio pode incluir backend, modelo, antena, potência, configurações do rotor e configurações UDP do WSJT-X.

<a id="wsjt-x-digital-operation"></a>
### Operação digital com WSJT-X

Quando a integração UDP do WSJT-X está habilitada, o CQOps pode receber mensagens ADIF do WSJT-X e registrar automaticamente QSOs digitais concluídos.

Os QSOs registrados automaticamente:

- são salvos no logbook ativo,
- aparecem imediatamente em Recent QSOs,
- ignoram duplicatas,
- herdam o ID do contest ativo,
- podem ser enviados automaticamente ao Wavelog quando ele estiver configurado e acessível.

Se o operador informado pelo WSJT-X não corresponder ao operador ativo no CQOps, o CQOps exibirá um aviso.

Antes de sessões digitais longas, confira:

- logbook ativo,
- operador ativo,
- contest ativo,
- rótulo de status do WSJT-X.

<a id="wavelog-sync"></a>
### Sincronização com Wavelog

O CQOps salva os QSOs localmente primeiro. A sincronização com Wavelog é opcional.

| Ação | Onde | Atalho | Observações |
|---|---|---|---|
| Enviar QSOs não enviados | Logbook Editor | `w` | Envia em lotes de 50 |
| Baixar do Wavelog | Logbook Editor | `Ctrl+W` | Download incremental usando `last_fetched_id` |

O status de envio é rastreado para cada QSO:

- não enviado,
- enviado,
- erro.

Se o envio falhar, o QSO permanece no logbook local e poderá ser reenviado posteriormente. Limpar completamente um logbook redefine o fetch ID para `0`, permitindo um novo download completo.

---

<a id="qso-logging"></a>
## Registro de QSOs

O formulário de QSO é a principal tela de registro. Abra-o com **F1**.

O CQOps pode preencher campos a partir de:

| Fonte | Campos |
|---|---|
| flrig / Hamlib | Frequência, Freq RX em split, modo, submodo |
| Callbook (QRZ.com / HamQTH / QRZ.RU / Callook.info) | Nome, QTH, grid, país, zona CQ, zona ITU, DXCC, continente, foto |
| Banco de dados REF | Referências SOTA, POTA, WWFF, IOTA |
| Consulta ao Wavelog | Status worked/confirmed quando configurado |
| Dados DXCC/prefixos | Prefixo e dados relacionados ao país |

<a id="form-layout"></a>
### Layout do formulário

| Coluna esquerda | Coluna central | Coluna direita |
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

Os campos de exchange aparecem somente quando há um contest ativo.

A linha inferior contém:

- **Comment**,
- **Keep** — mantém o campo Comment entre QSOs,
- **Retain** — mantém todo o formulário após salvar.

Campos como Band, Mode e Submode podem ser alternados com **PgUp/PgDn**.

<a id="path-bearing-and-badges"></a>
### Caminho, rumo e badges

Quando os dois grid locators são conhecidos, o CQOps mostra a distância e o azimute.

O formulário de QSO também pode mostrar badges como:

- **DUPE!**
- **New Call!**
- **New DXCC!**

<a id="saving"></a>
### Salvando

| Tecla | Ação |
|---|---|
| Enter | Salvar QSO |
| Ctrl+S | Enviar spot DX a partir do formulário preenchido |
| Esc | Cancelar a confirmação de duplicata |
| Enter na confirmação DUPE | Salvar a duplicata mesmo assim |

---

<a id="logbook-editor-and-adif"></a>
## Editor do logbook e ADIF

Abra o Logbook Editor com **F8**.

Use-o para:

- revisar QSOs,
- editar diretamente,
- excluir QSOs,
- importar ADIF,
- exportar ADIF,
- enviar ao Wavelog,
- baixar do Wavelog,
- operações relacionadas a contests.

<a id="editing-qsos"></a>
### Editando QSOs

1. Selecione uma linha com **↑/↓**.
2. Pressione **Enter** ou **e**.
3. Edite o QSO.
4. Salve com **Ctrl+S**.

As alterações aparecem imediatamente em Recent QSOs.

<a id="adif-import-and-export"></a>
### Importação e exportação ADIF

O CQOps oferece suporte à importação e exportação ADIF 3.1.7.

| Ação | Atalho |
|---|---|
| Importar ADIF | Ctrl+I |
| Exportar ADIF | Ctrl+E |

A importação valida os registros, ignora duplicatas e mostra um resumo. QSOs importados são marcados para envio ao Wavelog quando a sincronização com Wavelog está configurada.

A exportação pode incluir todos os QSOs ou somente QSOs filtrados por contest. O campo `CONTEST_ID` é preservado.

<a id="digital-mode-handling"></a>
### Tratamento dos modos digitais

O tratamento de modo e submodo segue o ADIF 3.1.7 conforme descrito neste manual:

- FT8 é exportado como um modo independente.
- FT4 e FT2 são exportados como MFSK com o submodo apropriado.
- Registros legados importados como MFSK + FT8 são normalizados para FT8 independente.

O formulário de QSO possui campos separados **Mode** e **Submode**. Ambos podem ser alternados com **PgUp/PgDn**.

---

<a id="contests"></a>
## Contests

O CQOps inclui um painel leve para registro de contests, projetado para **participação casual em contests** — ele não substitui loggers dedicados como N1MM, Win-Test ou TR4W. Para uma participação séria multi-op, multi-radio ou na categoria assisted, use um logger específico para contests. O CQOps é indicado quando você deseja distribuir alguns pontos, acompanhar sua taxa por diversão ou registrar alguns QSOs de contest durante uma ativação SOTA/POTA sem sair do logger usado no dia a dia.

<a id="setting-up-a-contest"></a>
### Configurando um contest

Crie ou configure um contest no Logbook Editor com **Ins**.

A configuração do contest inclui:

- nome do contest,
- data,
- ID ADIF do contest,
- templates de exchange.

<a id="template-markers"></a>
#### Marcadores de template

| Marcador | Substituído por |
|---|---|
| `@rst` | RST enviado ou recebido |
| `@serial` | Número de série incrementado automaticamente |
| `@cqz` | Zona CQ da estação DX |
| `@mycqz` | Sua zona CQ |
| `@itu` | Zona ITU da estação DX |
| `@myitu` | Sua zona ITU |
| `@grid` | Grid square da estação DX |
| `@mygrid` | Seu grid square |

Pressione **Ctrl+C** para alternar o contest ativo ou selecione-o no menu Contest (**F7**). Os campos de exchange aparecem automaticamente no formulário de QSO e os números de série são incrementados automaticamente.

<a id="bottom-status-bar"></a>
### Barra de status inferior

Quando um contest está ativo, a barra inferior mostra uma linha de resumo ao vivo:

```
 IARU-HF · IARU HF   45 QSOs   Started 16:13   Last 14:04 ago   Next #45   On 2:41
```

| Campo | Significado |
|-------|---------|
| `IARU-HF` | ID ADIF do contest, ou seja, o identificador legível por máquina |
| `· IARU HF` | Nome exibido do contest — mostrado quando é diferente do ID |
| `45 QSOs` | Total de QSOs registrados nesta sessão do contest |
| `Started 16:13` | Hora do primeiro QSO do contest no dia atual |
| `Last 14:04 ago` | Tempo desde o QSO mais recente do contest |
| `Next #45` | Número de série que será enviado no próximo QSO |
| `On 2:41` | Tempo total no ar — soma dos intervalos entre QSOs menores que 30 minutos |

O campo "Started" fica oculto em terminais estreitos, com menos de 120 colunas. O nome do contest e o tempo no ar ficam ocultos abaixo de 100 colunas.

<a id="contest-statistics-panel"></a>
### Painel de estatísticas do contest

Quando um contest está ativo e o terminal é suficientemente largo, um painel compacto de estatísticas aparece à direita do formulário de QSO, com borda amarela:

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

| Linha | Campo | Significado |
|-----|-------|---------|
| **Rate** | `2/h` | Taxa nos últimos **10 QSOs** — velocidade de curto prazo |
| | `--/h` | Taxa nos últimos **100 QSOs** — mostra `--` até que 100 QSOs sejam registrados |
| **Count** | `60m 0` | QSOs registrados nos últimos 60 minutos |
| | `hr 0` | QSOs registrados na hora atual do relógio, desde `:00` |
| **Peak** | `1m120` | Melhor taxa em 1 minuto: 120/h = 2 QSOs naquele minuto |
| | `10m 54` | Melhor janela móvel de 10 minutos: média de 54/h |
| | `60m 29` | Melhor janela móvel de 60 minutos: média de 29/h |
| **Avg** | `8/h` | Média da sessão — total de QSOs ÷ horas desde o primeiro QSO |
| | `Sess 5:36` | Duração total da sessão, do primeiro ao último QSO (H:MM ou somente minutos) |
| **Chart** | `max 1` | O minuto mais movimentado teve 1 QSO. As barras mostram QSOs por minuto |
| | `-60m…now` | Borda esquerda = 60 minutos atrás; borda direita = agora |

O gráfico usa caracteres de bloco Unicode (`█`) escalados para quatro linhas de barras verticais. As taxas Peak não mostram o sufixo `/h`, pois "Peak" já indica "por hora". Todas as durações omitem os segundos — em uma atualização por minuto, eles seriam apenas ruído.

<a id="contest-adif-export"></a>
### Exportação ADIF do contest

Para enviar seu log de contest, abra o **Logbook Editor** (`Ctrl+E`) enquanto um contest estiver ativo. Quando um filtro de contest estiver aplicado, a caixa de diálogo de exportação ADIF permite exportar **somente os QSOs pertencentes ao contest ativo**. Isso produz um arquivo ADIF 3.1.7 compatível com o padrão, preservando campos de exchange, números de série e o ID ADIF do contest — pronto para ser enviado ao robô do organizador ou ao sistema de verificação do log.

<a id="contest-mode-behavior"></a>
### Comportamento do modo contest

Quando um contest está ativo:

- o formulário de QSO mostra os campos de exchange,
- os números de série são incrementados automaticamente,
- Recent QSOs pode filtrar os QSOs do contest,
- a exportação ADIF preserva `CONTEST_ID`,
- o formulário de QSO, o painel do contest e o painel Solar recebem uma borda amarela para diferenciação visual,
- os spots DXC são comparados com todos os QSOs do contest, e não apenas os do dia atual, para marcação de duplicatas.

---

<a id="favorites-references-and-band-plans"></a>
## Favoritos, referências e planos de banda

<a id="favorites"></a>
### Favoritos

Os favoritos armazenam presets de frequência, modo e banda em três slots — o suficiente para as frequências de chamada mais usadas. Os atalhos utilizam `Alt` para evitar conflitos com teclas padrão de edição do terminal e funcionam de forma confiável em todos os tipos de terminal.

| Atalho | Ação |
|---|---|
| Alt+Ins / Alt+Home / Alt+PgUp | Recuperar o favorito do slot 1, 2 ou 3 |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Salvar a frequência, o modo e a banda atuais no slot 1, 2 ou 3 |

Os favoritos são armazenados na configuração e compartilhados entre os logbooks.

Exemplo:

1. Digite `145.55`.
2. Defina Mode como `FM`.
3. Defina Band como `2m`.
4. Pressione **Alt+Shift+Ins** para salvar no slot 1.
5. Posteriormente, pressione **Alt+Ins** para recuperar o preset.

<a id="ref-lookup"></a>
### Consulta REF

Abra REF Lookup com **F6**.

A busca inclui:

- SOTA,
- POTA,
- WWFF,
- IOTA.

Você pode buscar por prefixo, nome ou designador de referência. As referências selecionadas podem preencher o formulário de QSO.

<a id="band-plan-browser"></a>
### Navegador de planos de banda

Abra Band Plan Browser com **F7**.

Ele oferece acesso rápido a:

- bandas de radioamador,
- faixas VHF/UHF,
- CB,
- PMR446,
- presets de radiodifusão,
- Portable — frequências comuns para operação portátil/em campo, incluindo SOTA, POTA e canais de chamada.

Uma frequência selecionada pode ser usada para sintonizar o rádio ativo. Os dados do plano de banda também podem ser exportados como Markdown.

---
<a id="integrations"></a>
## Integrações

Todas as integrações são opcionais. O registro local funciona sem elas.

<a id="callbook-qrzcom-hamqth-callookinfo"></a>
### Callbook (QRZ.com, HamQTH, QRZ.RU, Callook.info)

O CQOps oferece suporte a vários provedores de callbook, consultados em cascata de acordo com a prioridade.
Quando você pressiona **Ins** no formulário de QSO, os provedores são consultados em ordem até que um deles retorne um resultado:

1. **QRZ.com** — exige internet e uma assinatura QRZ XML. Oferece os dados mais abrangentes.
2. **HamQTH** — serviço global gratuito. Boa cobertura; exige uma conta gratuita.
3. **QRZ.RU** — serviço gratuito voltado para a Rússia e países vizinhos. Requer login de API. Fornece nome, QTH, grid, lat/lon, classe, status LoTW/eQSL e foto.
4. **Callook.info** — serviço gratuito voltado aos Estados Unidos. Não exige conta e oferece consultas rápidas à FCC.

Se os provedores de prioridade mais alta falharem ou estiverem desabilitados, o próximo provedor será consultado.
Quando **Base call fallback** estiver habilitado (padrão: ativado), o CQOps também tentará consultar o indicativo-base, sem prefixo ou sufixo, caso não encontre o indicativo completo.

Habilite e configure os provedores em **F9 → Callbook**.

No formulário de QSO, pressione **Ins** para preencher campos do callbook, como:

- nome,
- QTH,
- grid,
- país,
- zonas CQ/ITU,
- DXCC,
- continente.

A Partner view em **F2** pode mostrar a foto do operador quando estiver disponível.

> ⚠️ **Experimental.** A exibição de fotos pode usar o protocolo gráfico do terminal Kitty
> e exige um terminal compatível: Kitty, Ghostty ou WezTerm.
> Habilite em **F9 → General → Kitty Graphics**. Terminais padrão e
> sessões SSH sem encaminhamento gráfico usarão como fallback uma foto em glifos.

<a id="wavelog"></a>
### Wavelog

A integração com Wavelog oferece suporte a:

- upload,
- download incremental,
- consulta de status worked/confirmed.

O Wavelog é configurado para cada logbook ativo com:

- URL,
- API key,
- station profile ID.

O CQOps sempre salva os QSOs localmente primeiro. Uma falha no upload para o Wavelog não exclui os dados locais.

<a id="flrig"></a>
### flrig

A integração com flrig usa XML-RPC sobre HTTP.

Endpoint padrão:

```text
localhost:12345
```

O CQOps pode ler:

- frequência,
- modo,
- potência.

Na operação em split, VFO A é mapeado para Frequency e VFO B para Freq RX.

<a id="hamlib--rigctld"></a>
### Hamlib / rigctld

O controle do rádio por Hamlib usa o daemon TCP `rigctld`.

Dependendo do rádio e do suporte do backend, o CQOps pode consultar:

- frequência,
- modo,
- VFO,
- split,
- potência.

O CQOps trata de forma adequada, sempre que possível, a ausência de suporte a nomes de VFO.

<a id="hamlib-rotator--rotctld"></a>
### Rotor Hamlib / rotctld

> ⚠️ **Experimental.** O controle do rotor é experimental. Sempre verifique os
> limites físicos da antena antes de operar. Esteja preparado para interromper
> imediatamente o movimento com **Alt+/**. Use com cuidado — uma configuração
> incorreta pode danificar o rotor ou a antena.

O controle do rotor usa o `rotctld` do Hamlib.

O CQOps oferece suporte a:

- azimute,
- elevação,
- comandos de parada.

| Atalho | Ação |
|---|---|
| Alt+, | Ajustar o azimute em −5° |
| Alt+. | Ajustar o azimute em +5° |
| Alt+; | Ajustar a elevação em +5° |
| Alt+' | Ajustar a elevação em −5° |
| Alt+\ | Apontar o rotor para o rumo calculado do caminho |
| Alt+/ | Parar o rotor |

<a id="wsjt-x"></a>
### WSJT-X

A integração com WSJT-X usa mensagens UDP do WSJT-X. O CQOps interpreta mensagens ADIF e pode registrar automaticamente QSOs concluídos.

O rótulo do rádio assume a cor de destaque enquanto o WSJT-X está transmitindo. Se o operador informado pelo WSJT-X não corresponder ao operador ativo, o CQOps mostrará um aviso.

<a id="gps"></a>
### GPS

O CQOps pode ler a posição de um receptor GPS e usá-la como grid locator da estação — ideal para operações portáteis, móveis ou em campo.

Dois backends são compatíveis:

- **Serial** — conecta-se diretamente a um receptor GPS por uma porta serial
  (adaptador USB-serial, porta COM integrada ou `/dev/ttyUSB0`).
- **GPSD** — conecta-se a um servidor [gpsd](https://gpsd.io/) por TCP
  (padrão `127.0.0.1:2947`). Útil quando o GPS é compartilhado com outros
  aplicativos ou acessado pela rede.

O indicador de status GPS na barra de status mostra:

| Cor | Significado |
|--------|---------|
| `GPS` vermelho | Desconectado / erro |
| `GPS` amarelo | Conectado, ainda sem fix |
| `GPS` branco | Fix obtido, posição bloqueada |

Quando um fix é obtido, o grid locator da estação é substituído pela posição
calculada a partir do GPS e marcado com `(GPS)` na linha de status:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Habilite **Grid from GPS** nas configurações de Station & Logbook para usar o
grid do GPS no registro de QSOs, nos beacons APRS, no mapa do dashboard e nos
cálculos de distância.

**Grid precision** — configurável no menu Integration em 10, 8 ou 6
caracteres. O padrão é 10 caracteres, com precisão aproximada de 25 m. O grid é
sempre calculado internamente com precisão total e truncado para o comprimento
configurado na camada de uso.

<a id="dx-cluster"></a>
### DX Cluster

A integração com DX Cluster usa telnet e exige acesso à internet.

Servidor padrão:

```text
dxspots.com:7300
```

Os filtros incluem:

- banda,
- continente do spotter,
- modo,
- idade/tempo.

| Tecla | Ação |
|---|---|
| Enter | Preencher o formulário de QSO, sintonizar o rádio e voltar ao QSO |
| Space | Sintonizar o rádio e permanecer no DX Cluster |
| Backspace | Limpar os filtros |

Quando o DX Cluster está conectado, o formulário de QSO recebe dois recursos adicionais:

- **Send a spot** — com o formulário preenchido, pressione **Ctrl+S** para abrir a caixa de diálogo de spot e enviar um spot DX ao cluster.
- **Nearest spots** — quando uma frequência está sintonizada, até três spots próximos aparecem diretamente no formulário de QSO, permitindo ver o que está na banda sem sair da tela de registro. Pressione **Ctrl+P** para preencher o indicativo a partir do spot mais próximo.

<a id="psk-reporter"></a>
### PSK Reporter

A integração com PSK Reporter exige acesso à internet. É uma excelente ferramenta para verificar rapidamente a propagação real — veja quem está recebendo seu sinal, ou quem você consegue ouvir, em qualquer banda e naquele momento.

Ela oferece:

- spots de propagação,
- filtros de banda/tempo/modo,
- mapa-múndi ASCII em **F5**.

<a id="aprs"></a>
### APRS

O CQOps oferece suporte a três tipos de serviço APRS — escolha aquele que corresponde à configuração de sua estação:

| Serviço | Conexão | Internet necessária |
|---|---|---|
| **APRS-IS** | TCP para um servidor APRS-IS | Sim |
| **KISS** | Porta serial para um TNC KISS físico | Não |
| **KISS Server** | TCP para um servidor TNC KISS, como o Dire Wolf | Não, quando está na rede local |

Selecione o tipo de serviço no menu Integrations:

```text
F9 → Integrations → APRS → Service (Space to cycle)
```

Os três serviços permitem receber relatórios de posição APRS de estações próximas
e exibi-los no mapa local do CQOps Live com:

- símbolos APRS padrão,
- pop-ups de indicativos,
- ajuste automático da visualização,
- círculo de alcance configurável.

Todos os serviços também oferecem **beaconing periódico de posição**. O CQOps transmite
o grid locator de sua estação no intervalo configurado. Quando o GPS está ativo
e **Grid from GPS** está habilitado, o beacon usa automaticamente a posição
obtida por GPS — ideal para operação portátil e móvel.

<a id="aprs-is"></a>
#### APRS-IS

Conecta-se à rede global APRS-IS pela internet. Exige:

- um indicativo de radioamador válido,
- um passcode APRS-IS gerado a partir de seu indicativo,
- uma conexão com a internet.

Servidor padrão:

```text
euro.aprs2.net:14580
```

O APRS-IS é configurado globalmente em **F9 → Integrations → APRS**.
Indicativo, SSID, símbolo, comentário, intervalo de beacon e filtro de alcance
específicos do logbook são configurados em **F9 → Logbooks → [active logbook] → APRS**.

<a id="kiss-serial"></a>
#### KISS (serial)

Conecta-se diretamente a um TNC KISS físico por uma porta serial. Não é necessária
uma conexão com a internet — os frames APRS são enviados e recebidos pelo rádio.

Configure a porta serial, baud rate, data bits, parity, stop bits e DTR/RTS no menu Integrations:

```text
F9 → Integrations → APRS → Service: KISS
```

Quando KISS está selecionado, os campos específicos da porta serial — Port, Baud,
Data bits, Parity, Stop bits, DTR e RTS — ficam visíveis.

O botão **Test** abre a porta serial para verificar se o TNC está acessível.

<a id="kiss-server-tcp"></a>
#### KISS Server (TCP)

Conecta-se a um TNC KISS acessível por TCP — por exemplo, uma instância do
[Dire Wolf](https://github.com/wb2osz/direwolf) executada no mesmo computador
ou na rede local. Não é necessária uma conexão com a internet.

Informe host e porta no menu Integrations:

```text
F9 → Integrations → APRS → Service: KISS Server → Host / Port
```

Padrão: `127.0.0.1:8001`

<a id="beaconing"></a>
#### Beaconing

Os beacons são enviados no intervalo configurado para cada logbook. O intervalo
mínimo é de 1 minuto. O beacon inclui:

- indicativo da estação com SSID,
- grid locator, obtido por GPS quando disponível,
- símbolo APRS,
- comentário opcional.

Quando o **GPS** está ativo e **Grid from GPS** está habilitado nas configurações
Station, o beacon usa automaticamente o grid locator obtido por GPS — não é
necessário atualizar manualmente o grid durante o deslocamento.

O intervalo de beacon e outras configurações específicas do logbook são definidos em:

```text
F9 → Logbooks → [active logbook] → APRS
```

<a id="receiving"></a>
#### Recepção

Os relatórios de posição APRS recebidos são armazenados em cache localmente e
exibidos no mapa do CQOps Live dashboard. As estações são mostradas com seus
símbolos APRS e podem ser clicadas para exibir detalhes. A visualização se ajusta
automaticamente para mostrar todas as estações visíveis dentro do alcance configurado.

A recepção APRS é independente da transmissão de beacons — você pode receber sem
enviar um beacon e vice-versa. Basta habilitar APRS no menu Integrations e definir
o tipo de serviço.

<a id="solar-data"></a>
### Dados solares

Os dados solares vêm de hamqsl.com e incluem:

- SFI,
- número de manchas solares,
- índices A/K,
- condições por banda.

As atualizações ao vivo exigem acesso à internet. Depois de uma busca bem-sucedida, os dados em cache permanecem disponíveis offline.

---

<a id="cqops-live-dashboard"></a>
## CQOps Live Dashboard

O CQOps Live é um dashboard integrado ao navegador para acompanhar a atividade da estação em tempo real.

Ele é útil para:

- telas públicas em field days,
- telas de estações de clube,
- monitoramento de contests,
- acompanhar a estação de outro cômodo,
- estandes em eventos ou feiras.

<a id="enable-the-dashboard"></a>
### Habilitar o dashboard

1. Pressione **F9**.
2. Abra **Integrations**.
3. Acesse **HTTP Server**.
4. Habilite **HTTP server**.
5. Opcionalmente, defina endereço e porta.
6. Pressione **Ctrl+S** para salvar.
7. Abra o dashboard em um navegador.

Configurações padrão:

| Configuração | Padrão |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

O servidor é iniciado imediatamente após salvar.

> **Vinculação de endereço:** o padrão `0.0.0.0` torna o dashboard acessível por qualquer dispositivo da rede local — útil para telas em field days, estações de clube ou para verificar a estação de outro cômodo. Defina Address como `127.0.0.1` para restringir o acesso somente ao computador local.

<a id="display-modes"></a>
### Modos de exibição

O CQOps Live possui dois modos de exibição.

<a id="overview-mode"></a>
#### Modo Overview

Exibido quando nenhum indicativo está sendo trabalhado ativamente.

Ele mostra:

- **mapas ao vivo** — marcadores dos QSOs de hoje com trajetórias de grande círculo entre o grid da estação e cada contato, além de um mapa APRS local mostrando as estações APRS próximas,
- tabela de Recent QSOs,
- informações da estação,
- estatísticas,
- acompanhamento das taxas de 5 minutos, 15 minutos e 1 hora,
- principais operadores,
- QSOs de maior distância.

<a id="active--now-working-mode"></a>
#### Modo Active / Now Working

Exibido quando um indicativo está sendo trabalhado.

Ele mostra:

- indicativo em tamanho grande,
- indicador de submodo,
- foto do QRZ quando disponível,
- badges de banda e modo,
- indicadores DUPE / NEW CALL / NEW DXCC,
- distância e rumo,
- trajetória tracejada destacada no mapa, do grid da estação ao grid do contato.

<a id="info-box"></a>
### Info box

A info box acima do mapa local alterna a cada 5 segundos entre os seguintes módulos:

- condições das bandas,
- atividade solar,
- campo geomagnético,
- spot mais recente do DX Cluster,
- número de reports por banda no PSK Reporter.

<a id="weather-row"></a>
### Linha de meteorologia

A linha de meteorologia mostra as condições atuais do Open-Meteo para o grid locator da estação:

- temperatura,
- vento,
- umidade,
- ícone.

Os dados meteorológicos são obtidos pelo navegador e o recurso se degrada de forma adequada quando está offline.

<a id="local-map"></a>
### Mapa local

O mapa local do lado direito é dedicado ao **monitoramento da vizinhança APRS** — veja quem está no APRS ao redor de sua estação. Ele pode mostrar:

- estações APRS próximas com símbolos APRS padrão,
- pop-ups de indicativos ao passar o mouse ou clicar,
- círculo de alcance configurável,
- sobreposição opcional do terminador dia/noite,
- sobreposição opcional do radar meteorológico RainViewer.

<a id="real-time-updates-and-performance"></a>
### Atualizações em tempo real e desempenho

O CQOps Live é atualizado por Server-Sent Events (SSE). Não é necessário atualizar a página.

O dashboard foi projetado para hardware de baixo consumo:

- o navegador renderiza os mapas,
- o navegador calcula as distâncias,
- o navegador calcula as estatísticas,
- o CQOps envia atualizações JSON leves,
- quando o HTTP server está desabilitado, nenhuma porta é aberta e as goroutines do dashboard não são executadas.

<a id="dashboard-customization"></a>
### Personalização do dashboard

No formulário da integração HTTP Server, você pode configurar:

| Campo | Descrição |
|---|---|
| Header 1 | Título principal mostrado no cabeçalho e na área de destaque da página. Usa “CQOps Live” como fallback. |
| Header 2 | Subtítulo abaixo do título. Usa “Fast, portable ham radio logger” como fallback. |
| Logo URL | URL de uma imagem acessível publicamente, mostrada no canto superior esquerdo. Usa o logotipo do CQOps como fallback. |
| Event Start | Data no formato `YYYY-MM-DD`. Filtra estatísticas e listas de QSOs a partir dessa data. |

---
<a id="configuration"></a>
## Configuração

Abra a configuração com **F9**.

<a id="configuration-files"></a>
### Arquivos de configuração

| Plataforma | Caminho da configuração |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Credenciais confidenciais são armazenadas separadamente em `secrets.enc`, no mesmo diretório da configuração.

Os secrets são criptografados com uma chave vinculada ao computador. Ao mover a configuração para outro computador, as credenciais precisam ser informadas novamente.

<a id="configuration-menus"></a>
### Menus de configuração

Pressione **F9** para abrir o menu principal e selecione:

| Menu | O que configura |
|---|---|
| General | Unidades, fuso horário, mapa/foto do contato, painel Solar, fontes de dados SCP/REF, gráficos Kitty, debug mode |
| Logbooks | Indicativo da estação, grid, referências, zonas CQ/ITU, região IARU, grid GPS; Wavelog por logbook (URL, API key, station profile); APRS por logbook (indicativo, símbolo, beacon, alcance) |
| Operators | Perfis com indicativo e nome do operador para estações multioperador |
| Rigs | Presets de rádio: modelo, antena, potência, backend (None/flrig/Hamlib), rotor, UDP do WSJT-X |
| Contests | Perfis de contest: nome, data, ID ADIF do contest, templates de exchange, número de série inicial |
| Integration | DX Cluster, HTTP server do dashboard, GPS, APRS, Solar, PSK Reporter |
| Callbook | Provedores QRZ.com, HamQTH, QRZ.RU e Callook.info; ordem de prioridade, base-call fallback, consulta ao Wavelog |
| Notifications | Alertas de QSO salvo, status de QSO enviado ao Wavelog, beep de duplicata, sons de erro |

<a id="multi-logbook"></a>
### Vários logbooks

Use vários logbooks para operação em casa, portátil, em contests e em clubes.

Pressione **Ctrl+L** para alternar o logbook ativo.

Cada logbook mantém seus próprios:

- dados da estação,
- configurações do Wavelog,
- configurações de contest,
- configurações de operadores.

<a id="multi-operator"></a>
### Vários operadores

Os perfis de operador contêm:

- indicativo do operador,
- nome do operador.

Pressione **Ctrl+O** para alternar o operador ativo.

O operador ativo é salvo no campo ADIF `OPERATOR` e acompanha os uploads para o Wavelog.

<a id="multi-rig"></a>
### Vários rádios

Os presets de rádio armazenam:

- backend,
- modelo,
- antena,
- potência,
- configurações do rotor,
- configurações do WSJT-X.

Pressione **Ctrl+R** para alternar o rádio ativo.

<a id="encrypted-secrets"></a>
### Secrets criptografados

Desde a versão v0.8.7, as credenciais são armazenadas de forma criptografada.

| Item | Valor |
|---|---|
| Arquivo de secrets | `secrets.enc` |
| Localização | Mesmo diretório de `config.yaml` |
| Permissões Unix | `0600`, onde houver suporte |
| Criptografia | AES-256-GCM com uma chave vinculada ao computador |
| Dados protegidos | Senha do QRZ, login do DX Cluster, API keys do Wavelog |

Secrets em texto simples de configurações antigas são migrados na primeira execução.

Se `secrets.enc` estiver corrompido, o CQOps iniciará com um aviso e solicitará que você informe novamente as credenciais.

---

<a id="keyboard-shortcuts"></a>
## Atalhos de teclado

<a id="global"></a>
### Globais

| Tecla | Ação |
|---|---|
| F1 | QSO form e Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF Lookup |
| F7 | Band Plan Browser |
| F8 | Logbook Editor |
| F9 | Configuration / main menu |
| F10 | Sair |
| Ctrl+F9 | Log viewer |
| ? | Help overlay |
| Ctrl+L | Alternar o logbook ativo |
| Ctrl+R | Alternar o rádio ativo |
| Ctrl+C | Alternar o contest ativo |
| Ctrl+O | Alternar o operador ativo |
| Esc | Voltar à tela anterior |

<a id="qso-form"></a>
### Formulário de QSO

| Tecla | Ação |
|---|---|
| Tab | Próximo campo |
| Shift+Tab | Campo anterior |
| ↑ / ↓ | Mover dentro da coluna |
| Enter | Salvar QSO, com confirmação de duplicata quando necessário |
| Del | Limpar todos os campos do formulário |
| Ins | Lookup: Callbook, Wavelog, DXCC e verificação de duplicata |
| PgUp / PgDn | Alternar banda, modo ou submodo |
| Ctrl+S | Enviar spot DX a partir do formulário preenchido |
| Ctrl+P | Preencher o indicativo a partir do spot DXC mais próximo |
| Ctrl+C | Alternar o contest ativo |
| Alt+, | Ajustar o azimute do rotor em −5° |
| Alt+. | Ajustar o azimute do rotor em +5° |
| Alt+; | Ajustar a elevação do rotor em +5° |
| Alt+' | Ajustar a elevação do rotor em −5° |
| Alt+\ | Apontar o rotor para o rumo entre o próprio grid e o grid do contato |
| Alt+/ | Parar o rotor |
| Alt+Ins / Alt+Home / Alt+PgUp | Recuperar favorito (slot 1/2/3) |
| Alt+Shift+Ins / Alt+Shift+Home / Alt+Shift+PgUp | Salvar frequência, modo e banda em um favorito |

<a id="logbook-editor"></a>
### Logbook Editor

| Tecla | Ação |
|---|---|
| ↑ / ↓ | Navegar pelas linhas |
| PgUp / PgDn | Página anterior ou seguinte |
| Home / End | Primeira ou última linha |
| Enter / e | Editar o QSO selecionado |
| Delete | Excluir o QSO selecionado |
| p | Remover todos os QSOs |
| Ctrl+C | Alternar o filtro de contest |
| Ctrl+E | Exportar ADIF |
| Ctrl+I / Tab | Importar ADIF |
| w | Enviar QSOs não enviados ao Wavelog |
| Ctrl+W | Baixar contatos do Wavelog |
| Esc / F6 | Fechar o editor e voltar ao formulário de QSO |

<a id="dx-cluster-shortcuts"></a>
### DX Cluster

| Tecla | Ação |
|---|---|
| ↑ / ↓ | Navegar pelos spots |
| Enter | Preencher o formulário de QSO, sintonizar o rádio e voltar ao QSO |
| Space | Sintonizar o rádio no spot selecionado e permanecer no DX Cluster |
| Home | Avançar pelo filtro de banda |
| End | Retroceder pelo filtro de banda |
| `\` | Alternar o filtro de continente do spotter |
| Ins | Avançar pelo filtro de modo |
| Del | Retroceder pelo filtro de modo |
| PgUp | Avançar pelo filtro de tempo |
| PgDn | Retroceder pelo filtro de tempo |
| Backspace | Limpar todos os filtros |
| Esc / F4 | Voltar ao formulário de QSO |

<a id="partner-view"></a>
### Partner view

| Tecla | Ação |
|---|---|
| F2 | Alternar Partner view → Photo → Back |
| Esc / F1 | Voltar ao formulário de QSO |

---

<a id="troubleshooting"></a>
## Solução de problemas

<a id="cqops-does-not-start"></a>
### O CQOps não inicia

Verifique se:

- o terminal tem pelo menos 80×24,
- usuários do Windows estão usando Windows Terminal,
- a inicialização da rede não está bloqueando o aplicativo; teste:

  ```bash
  cqops --offline
  ```

Verifique os logs:

| Plataforma | Caminho dos logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

<a id="rig-does-not-connect"></a>
### O rádio não conecta

Para flrig:

- verifique se o flrig está em execução,
- verifique a porta no preset de rádio ativo,
- a porta padrão é `12345`.

Para Hamlib:

- verifique se o `rigctld` está em execução,
- verifique host e porta,
- confira se o rádio/backend oferece suporte aos dados solicitados.

Os rótulos de status ajudam a diagnosticar o problema:

| Cor | Significado |
|---|---|
| Branco/padrão | Conectado |
| Amarelo | Desabilitado ou conectando |
| Vermelho | Falha |

Toasts de reconexão podem ser suprimidos. O CQOps pode tentar reconectar silenciosamente.

<a id="wsjt-x-does-not-auto-log"></a>
### O WSJT-X não registra automaticamente

Verifique:

- **Settings → Reporting → UDP Server** no WSJT-X,
- se host e porta UDP correspondem ao preset de rádio ativo no CQOps,
- se está sendo usado o WSJT-X 2.6 ou mais recente,
- se o rótulo de status WSJT está ativo,
- se o logbook ativo está correto,
- se o operador ativo está correto.

<a id="wavelog-upload-fails"></a>
### O upload para o Wavelog falha

Verifique:

- URL do Wavelog,
- API key,
- station profile ID,
- rótulo de status **WL**.

Erros de upload são mostrados como toasts. Os QSOs permanecem salvos localmente mesmo quando o upload falha. Falhas em QSOs individuais não bloqueiam o restante do lote.

<a id="config-file-issues"></a>
### Problemas com o arquivo de configuração

Arquivo de configuração:

| Plataforma | Caminho |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Arquivo de secrets:

```text
secrets.enc
```

O arquivo de secrets é armazenado no mesmo diretório de `config.yaml`.

Se a configuração estiver corrompida, mova-a ou exclua-a e reinicie o CQOps. O assistente de configuração criará uma nova configuração.

O campo `last_fetched_id` aparece somente depois de um download bem-sucedido do Wavelog.

<a id="performance-issues"></a>
### Problemas de desempenho

Tente:

- desabilitar a renderização do mapa nas configurações General,
- desabilitar o painel Solar se não for necessário,
- evitar telas que usam intensamente a rede, como DX Cluster e PSK Reporter, quando estiver offline,
- usar `cqops --offline` quando a rede não for confiável.

---

<a id="reporting-bugs"></a>
## Relatando bugs

Antes de relatar um bug:

1. Habilite **Debug mode** em **F9 → General → Debug** ou defina:

   ```yaml
   debug: true
   ```

   em `config.yaml`.

2. Reproduza o problema.
3. Anexe o log relevante.

Relate problemas no GitHub:

<https://github.com/szporwolik/cqops/issues>

Inclua:

- versão do CQOps obtida com `cqops --version`,
- sistema operacional,
- emulador de terminal,
- passos para reproduzir,
- log de debug relevante.
