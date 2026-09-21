---
title: Manual do usuário do CQOps
description: Guia prático para configurar o CQOps e registrar contatos na estação ou em campo
---

# Manual do usuário do CQOps

O CQOps é um programa de registro de contatos de radioamador operado pelo teclado, para uso em casa, em campo, em estações de clube e em concursos ocasionais. Os QSOs são salvos primeiro no computador; os serviços de internet são opcionais. Comece pelo registro manual e adicione controle de rádio ou serviços on-line quando precisar.

Os nomes de menus e campos abaixo correspondem à interface em inglês. Os atalhos dependem da tela ativa: confira a barra de ajuda inferior ou pressione **?**.

## Sumário

1. [Instalação](#installation)
2. [Configuração inicial](#setup)
3. [Primeiro QSO](#first-qso)
4. [Telas e status](#screens)
5. [Registro no dia a dia](#logging)
6. [Perfis de estação](#profiles)
7. [Livro de registro e backups](#logbook)
8. [Rádio e modos digitais](#radio)
9. [Serviços on-line](#online)
10. [GPS e APRS](#position)
11. [Operação em campo](#portable)
12. [Concursos](#contests)
13. [CQOps Live](#dashboard)
14. [Atalhos de teclado](#keys)
15. [Solução de problemas e ajuda](#help)

<a id="installation"></a>

## Instalação

Baixe o CQOps na [página de versões](https://github.com/szporwolik/cqops/releases). O terminal precisa ter pelo menos 75 × 24 caracteres; 80 × 43 ou mais é mais confortável.

| Sistema | Instalação |
|---|---|
| Windows | Baixe `cqops-setup.exe` ou extraia `cqops-windows-portable.zip` para usar sem instalar. Recomendamos o Windows Terminal. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Baixe o `.deb` adequado: `amd64` para a maioria dos PCs Intel/AMD, `arm64` para ARM de 64 bits ou `armhf` para Raspberry Pi OS de 32 bits. Abra no instalador de pacotes. |
| Fedora, RHEL, Rocky, AlmaLinux | Use os comandos do repositório abaixo. |
| Arch, Manjaro, CachyOS | Instale o pacote AUR com `paru -S cqops-bin` ou `yay -S cqops-bin`. |
| Outros sistemas Linux | Baixe e extraia o arquivo Linux `.tar.gz` correspondente ao processador. |
| macOS | Baixe `cqops-darwin-arm64` para Apple Silicon ou `cqops-darwin-amd64` para Intel. Use os comandos abaixo. |

Em sistemas baseados no Debian, você também pode instalar pelo repositório:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

Em sistemas baseados no Fedora:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

No macOS, execute na pasta de download, substituindo `FILE` pelo nome exato do arquivo baixado:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Inicie com `cqops` ou execute o programa portátil extraído. `cqops --offline` inicia sem rede, `cqops --version` mostra a versão e `cqops --help` lista opções de inicialização. Exporte seus registros antes de atualizar.

<a id="setup"></a>

## Configuração inicial

O assistente solicita nome do livro de registro, indicativo da estação, localizador Maidenhead e continente. O **indicativo da estação** é o usado na transmissão; o **perfil do operador** identifica a pessoa operando a estação.

**Ctrl+A** mostra referências de estação e zonas CQ/ITU opcionais. Sua referência SOTA/POTA/WWFF pertence às configurações da estação/livro; as referências do formulário QSO são do correspondente. Configure depois a região IARU em **F9 → Logbooks**.

Crie um perfil de rádio com nome, antena e potência. Escolha **None** para informar frequência e modo manualmente, **flrig** ou **Hamlib**. Configure conexões opcionais depois de testar o registro básico.

**Tab / Shift+Tab** muda de campo, **Space** altera opções e **Save & Next** avança. **Esc** volta; **F10** sai. Confira o resumo e salve. O CQOps detecta o fuso horário do computador; datas e horários dos QSOs usam UTC. Confira o relógio antes de operar.

<a id="first-qso"></a>

## Primeiro QSO

1. Pressione **F1**. Confira livro, indicativo, operador, rádio e concurso ativos.
2. Digite o indicativo do correspondente. **Ins** faz a consulta, se configurada.
3. Confira data/hora UTC, frequência em MHz, banda, modo e reportagens enviadas/recebidas.
4. Acrescente nome, QTH, localizador, referência ou comentário se necessário.
5. Pressione **Enter**. O contato aparece em Recent QSOs.

Se **DUPE!** pedir confirmação, pressione **Enter** novamente para salvar mesmo assim, ou **Esc** para cancelar a confirmação. O aviso pede uma verificação; não significa que o contato deva ser descartado.

<a id="screens"></a>

## Telas e status

| Tecla | Tela | Uso |
|---|---|---|
| F1 | QSO | Registrar contatos e ver QSOs recentes |
| F2 | Partner | Dados do correspondente, mapa, estatísticas, foto |
| F3 | APRS | Estações próximas |
| F4 | DX Cluster | Spots e filtros |
| F5 | PSK Reporter | Relatórios de recepção digital |
| F6 | References | Pesquisa SOTA, POTA, WWFF, IOTA |
| F7 | Band Plan | Frequências e predefinições |
| F8 | Logbook | Editar, importar, exportar, sincronizar |
| F9 | Configuration | Configurações da estação e serviços |
| F10 | Quit | Sair do CQOps |

A barra superior mostra a configuração ativa, hora local (**L**) e UTC (**Z**). Branco normalmente indica ativo, amarelo desativado/conectando/aguardando e vermelho erro. WSJT fica destacado durante a transmissão. **WL!** indica uma chave antiga do Wavelog não suportada.

<a id="logging"></a>

## Registro no dia a dia

Use **Tab / Shift+Tab** entre os campos e **PgUp / PgDn** para alternar banda, modo ou submodo. **Shift+Backspace** limpa o campo; **Del** limpa o formulário. Confira **Freq RX** ao operar em split.

**Keep** mantém o comentário após salvar. **Retain** mantém o formulário inteiro: confira indicativo, horário, reportagens e referências antes do próximo contato. Campos de intercâmbio aparecem apenas com concurso ativo. Use **SIG / SIG Info** para outras informações de grupos de interesse especial.

Com os dois localizadores conhecidos, o CQOps mostra distância e azimute. O callbook pode indicar a estação de casa, não a posição portátil atual; verifique. Indicadores de novo indicativo, novo DXCC e duplicata ajudam a avaliar o contato.

**F6** pesquisa referências por nome ou código e pode preencher a referência do correspondente. **F7** mostra planos de banda e pode sintonizar um rádio conectado. São auxílios operacionais, não autorização para transmitir: confira suas permissões e o plano local.

Três favoritos compartilhados guardam frequência, modo e banda:

| Posição | Recuperar | Salvar valores atuais |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Perfis de estação

Crie livros, operadores, rádios e concursos nos respectivos menus **F9**; **Ins** adiciona uma entrada. Na tela QSO:

| Atalho | Alternar |
|---|---|
| Ctrl+L | Livro de registro |
| Ctrl+O | Operador |
| Ctrl+R | Rádio |
| Ctrl+C | Concurso |

Cada livro mantém dados da estação e configurações Wavelog/APRS separados. O perfil de operador identifica a pessoa; o indicativo é gravado no campo ADIF `OPERATOR`. Perfis de rádio guardam equipamento, potência, controle de rádio/rotor e WSJT-X. Confira o status após cada troca, principalmente no registro digital automático.

Outros menus **F9** tratam de exibição, unidades, fuso horário, callbooks, integrações e avisos sonoros.

<a id="logbook"></a>

## Livro de registro e backups

Em **F8**, selecione um QSO e pressione **Enter** ou **e** para editar. Salve com **Enter** e confirme. **Delete** exclui o contato selecionado. Faça backup antes de mudanças em massa; **Ctrl+P** apaga todos os QSOs, não é pesquisa.

| Atalho F8 | Ação |
|---|---|
| Ctrl+I | Importar ADIF, validar registros e ignorar duplicatas |
| Ctrl+E | Exportar todos os contatos ou uma seleção por concurso |
| Ctrl+W | Enviar contatos pendentes ao Wavelog |
| Alt+W | Baixar do Wavelog |

Confira o resumo de importação e a seleção de exportação. Contatos importados podem ser enviados depois ao Wavelog. O CQOps suporta ADIF 3.1.7 e preserva IDs de concursos e intercâmbios. Guarde um backup por livro, de preferência em outro dispositivo. ADIF protege os contatos, não todas as configurações e credenciais.

A configuração fica em `~/.config/cqops/config.yaml` no Linux/macOS e `%APPDATA%\cqops\config.yaml` no Windows. Credenciais ficam separadas em `secrets.enc`; informe-as novamente ao mudar de computador. Não comece um diagnóstico apagando a configuração.

<a id="radio"></a>

## Rádio e modos digitais

### Controle do rádio

Em **F9 → Rigs**, selecione flrig ou Hamlib e ajuste a conexão. Inicie primeiro flrig ou `rigctld`. flrig normalmente usa `localhost:12345`. Leituras de frequência, modo, split e potência dependem do rádio. Com **None**, informe manualmente.

### WSJT-X

Use WSJT-X 2.6 ou posterior. Faça **Settings → Reporting → UDP Server** corresponder aos parâmetros UDP do perfil CQOps ativo. Registre um QSO de teste concluído no WSJT-X e confirme que aparece no CQOps.

QSOs recebidos usam livro e concurso ativos; duplicatas são ignoradas. Confira operador e indicador WSJT antes da sessão. O CQOps avisa se os operadores não coincidem. O envio ao Wavelog pode ocorrer quando configurado. Escolha Mode/Submode corretamente: FT8 é exportado como FT8; FT4/FT2 como MFSK com o submodo correspondente.

### Controle do rotor

O controle por Hamlib `rotctld` é experimental. Verifique direção e limites físicos antes de usar. Tenha uma forma segura de parar: configurações erradas podem danificar antena, rotor ou linha de alimentação.

| Atalho | Ação |
|---|---|
| Alt+, / Alt+. | Azimute −5° / +5° |
| Alt+' / Alt+; | Elevação −5° / +5° |
| Alt+\ | Apontar para o azimute calculado |
| Alt+/ | Parar movimento |

<a id="online"></a>

## Serviços on-line

### Callbooks

Configure provedores e prioridade em **F9 → Callbook**, depois pressione **Ins** no formulário QSO. Os provedores habilitados são consultados em ordem. A busca pelo indicativo base pode remover prefixos/sufixos portáteis; confira a localização retornada.

| Provedor | Acesso |
|---|---|
| QRZ.com | Assinatura XML e credenciais |
| HamQTH | Conta gratuita |
| QRZ.RU | Login de API separado do acesso ao site |
| Callook.info | Indicativos dos EUA; sem conta |

**F2** mostra o correspondente. Fotos dependem do provedor e terminal; **Kitty Graphics**, experimental em General, exige um terminal compatível, como Kitty, Ghostty ou WezTerm.

### Wavelog

Configure URL, token API v2 (`wl2_…`) e perfil de estação por livro. Chaves v1 antigas não são aceitas. Escolher uma estação Wavelog pode preencher dados locais: confira indicativo, localizador e referências antes de salvar.

QSOs são salvos primeiro localmente. Repita envios com falha em **F8 → Ctrl+W**; **Alt+W** baixa contatos. Ao abrir um QSO vinculado para editar, o CQOps pode atualizá-lo a partir do Wavelog. Edições e exclusões on-line afetam também a cópia remota. Leia a confirmação, especialmente off-line; não presuma que uma mudança apenas local chegou ao Wavelog.

### DX Cluster e propagação

Configure DX Cluster em Integrations e abra **F4**. **b / c / m / t** filtram banda, continente de quem publicou o spot, modo e idade. **Backspace** limpa os filtros. **Enter** preenche QSO, sintoniza o rádio conectado e volta a F1; **Space** sintoniza sem sair do cluster.

Em F1, **Ctrl+S** abre a janela de spot e **Ctrl+P** usa o indicativo do spot exibido mais próximo. Confira antes de enviar. **F5** mostra relatórios PSK Reporter, não garante a propagação atual. Solar mostra condições HamQSL; os dados em cache podem estar antigos.

<a id="position"></a>

## GPS e APRS

### GPS

Configure GPS serial ou GPSD em Integrations. Ative **Grid from GPS** nas configurações da estação/livro para usar o localizador em QSOs, azimutes, APRS e painel. GPS vermelho indica erro; amarelo, sem posição; branco, posição obtida. Confira o localizador antes de operar. Escolha 6, 8 ou 10 caracteres; mais caracteres não garantem maior precisão do receptor.

### APRS

| Serviço | Conexão |
|---|---|
| APRS-IS | Servidor APRS na internet |
| KISS | TNC físico serial e rádio |
| KISS Server | TNC TCP como Dire Wolf; pode ser local |

Escolha em **F9 → Integrations → APRS**. Ajuste indicativo/SSID, símbolo, comentário, alcance e intervalo em **F9 → Logbooks → [active logbook] → APRS**. Ative **APRS TX** e **Send beacons** somente se quiser transmitir. A recepção exclusiva mostra **APRS-RX**. Beacons revelam sua posição: confira-a e considere quem poderá recebê-la.

O intervalo automático mínimo é cinco minutos. **F3** mostra estações ouvidas recentemente: setas selecionam, **Enter** preenche QSO, **d / t / s** filtram distância/idade/tipo, **Backspace** limpa filtros e **b** envia um beacon configurado imediatamente. Beacons GPS exigem **Grid from GPS** e posição válida.

<a id="portable"></a>

## Operação em campo

Antes de sair, selecione o livro portátil; confira indicativo, localizador, referência da ativação, rádio, antena e potência. Teste a estação completa e execute CQOps on-line para atualizar referências e prefixos. Confira se **F6** encontra as referências necessárias. Exporte um backup.

O registro local funciona sem internet. `cqops --offline` ignora funções de rede; não dependa de consultas ao vivo nem sincronização. Teste equipamentos na rede local com o modo de inicialização escolhido antes de sair. Dados em cache podem estar desatualizados.

Depois, confira quantidade de QSOs e referências, exporte ADIF, guarde uma cópia e envie contatos pendentes ao Wavelog, se usado. Verifique o formato exigido por cada programa de diplomas e converta quando necessário.

<a id="contests"></a>

## Concursos

O CQOps oferece registro ocasional de concursos, intercâmbios, números sequenciais e ritmo de QSOs. Não é um sistema completo de pontuação ou envio de logs. Para operação avançada, use um programa especializado.

Em **F9 → Contests**, pressione **Ins** e defina nome, data, ID ADIF do concurso, número inicial e modelos de intercâmbio enviado/recebido.

| Marcador | Valor |
|---|---|
| `@rst` | Reportagem enviada ou recebida |
| `@serial` | Número sequencial |
| `@cqz` / `@mycqz` | Zona CQ do contato / sua |
| `@itu` / `@myitu` | Zona ITU do contato / sua |
| `@grid` / `@mygrid` | Localizador do contato / seu |

Em **F1**, **Ctrl+C** alterna concursos. Confira intercâmbio e próximo número antes de transmitir. A barra mostra total, próximo número e tempos; janelas maiores mostram mais estatísticas de ritmo. Ao terminar, volte à operação sem concurso ativo.

Para exportar, abra **F8**, selecione o filtro com **Ctrl+C**, depois **Ctrl+E** e confira a seleção. O resultado é ADIF, não Cabrillo. Siga formato e regras de envio do organizador.

<a id="dashboard"></a>

## CQOps Live

Ative **F9 → Integrations → HTTP Server** e salve com **Ctrl+S**. No computador CQOps, abra `http://localhost:8073`.

O endereço padrão `0.0.0.0` permite acesso na rede local, conforme o firewall. Em outro dispositivo use o IP do computador CQOps e a porta `8073`. `127.0.0.1` restringe ao próprio computador. Use uma rede confiável; não redirecione a porta para a internet.

O painel atualiza contato atual, mapas QSO, contatos recentes, ritmo, operadores, APRS e dados disponíveis de propagação/clima. Camadas dependentes de internet podem faltar off-line. Header 1, Header 2, Logo URL e Event Start personalizam o evento; a data inicial filtra estatísticas e listas de QSOs.

<a id="keys"></a>

## Atalhos de teclado

| Tela | Teclas | Ação |
|---|---|---|
| Geral | ? / Esc / F10 | Ajuda / voltar / sair |
| QSO | Tab / Shift+Tab | Próximo campo / anterior |
| QSO | Enter / Ins | Salvar / consultar |
| QSO | Shift+Backspace / Del | Limpar campo / formulário |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Alternar livro / operador / rádio / concurso |
| Livro | ↑ / ↓, PgUp / PgDn, Home / End | Seleção, página, primeira / última linha |
| Livro | Enter ou e / Delete | Editar / excluir QSO selecionado |
| Livro | Ctrl+I / Ctrl+E | Importar / exportar ADIF |
| Livro | Ctrl+W / Alt+W | Enviar / baixar Wavelog |
| Livro | Ctrl+C / Backspace | Filtro de concurso / limpar pesquisa |

Os atalhos dependem da tela: **Ctrl+C não fecha o CQOps**. Em notebooks, as teclas de função podem exigir **Fn**. Se o terminal capturar um atalho, confira suas configurações de teclado e a barra de ajuda CQOps.

<a id="help"></a>

## Solução de problemas e ajuda

| Problema | Verificar primeiro |
|---|---|
| Inicialização ou tela incompleta | Tamanho do terminal, Windows Terminal no Windows, testar `cqops --offline` |
| Rádio desconectado | Perfil ativo, flrig/rigctld iniciado, modelo, porta serial, velocidade, host/porta, outro programa ocupando a serial |
| QSO WSJT-X ausente | UDP compatível, indicador WSJT, QSO realmente salvo no WSJT-X, livro ativo |
| Erro Wavelog | URL, token `wl2_`, perfil, internet; QSOs locais permanecem salvos |
| Sem posição GPS | Porta/velocidade ou endereço GPSD, céu livre, posição válida, Grid from GPS |
| APRS sem beacons | Livro, APRS TX e Send beacons, indicativo/SSID, TNC/rádio ou internet |
| Painel inacessível | Servidor ativo, IP/porta corretos, firewall; localhost indica o dispositivo do navegador |

Use **F9** antes de editar arquivos de configuração. Reinsira credenciais se houver um aviso sobre segredos salvos ou após mudar de computador.

Se necessário, ative **F9 → General → Debug**, reproduza o problema com segurança e recolha o log diagnóstico; desative o debug depois.

| Sistema | Logs de diagnóstico |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Relate problemas em [GitHub Issues](https://github.com/szporwolik/cqops/issues), incluindo versão CQOps, sistema, terminal, passos e log relevante. Remova senhas, tokens API e informações privadas antes de compartilhar.
