# Instalação do ERP Venture no Windows

Este guia orienta a instalação oficial do aplicativo desktop e a primeira conexão com o VentureERP. Não baixe executáveis enviados por e-mail, mensageiro ou sites diferentes do repositório oficial.

## Requisitos

- Windows 10 versão 1809 ou Windows 11, 64 bits, atualizado;
- acesso HTTPS a `api.venturerp.com` e `github.com`;
- permissão para instalar aplicativos no computador;
- credenciais fornecidas pelo administrador do ERP;
- Microsoft Edge WebView2 Runtime. Windows atualizado normalmente já o possui.

## Baixar o instalador

1. Abra a página oficial **Releases** do ERP Venture Desktop no GitHub.
2. Escolha a release mais recente marcada como estável, com versão no formato `vX.Y.Z`.
3. Em **Assets**, baixe preferencialmente o instalador NSIS `.exe`. O pacote `.msi` é destinado à distribuição corporativa por TI.
4. Não baixe `latest.json` nem arquivos `.sig`; eles são usados automaticamente pelo atualizador.
5. Confira se o nome e a versão pertencem à mesma release. Não renomeie nem modifique o instalador.

## Instalar com NSIS

1. Feche uma instalação anterior do ERP Venture.
2. Execute o `.exe` baixado.
3. Se o Windows mostrar SmartScreen, confirme que o arquivo veio da release oficial. A assinatura criptográfica do updater protege atualizações internas; um aviso de reputação do Windows pode existir enquanto não houver certificado Authenticode comercial.
4. Escolha **Mais informações > Executar assim mesmo** somente quando a origem oficial tiver sido confirmada.
5. Aceite o diretório sugerido e conclua a instalação.
6. Abra **ERP Venture** pelo menu Iniciar.

## Instalação corporativa com MSI

Um administrador pode instalar o `.msi` em terminal elevado:

```powershell
msiexec /i "ERP.Venture_X.Y.Z_x64.msi" /passive /norestart
```

Para implantação por Intune/GPO, distribua exatamente o MSI da GitHub Release e valide primeiro em um grupo de homologação. Não reempacote o executável, pois isso prejudica rastreabilidade e suporte.

## Primeiro acesso

1. Mantenha conexão com a internet e abra o aplicativo.
2. O app consulta `https://api.venturerp.com/api/version`. Se não conseguir validar o servidor, ele não libera uma sessão potencialmente incompatível e oferece **Tentar novamente**.
3. Se uma versão mínima mais nova for obrigatória, clique em **Instalar** e aguarde reinício automático.
4. Na tela de login, informe e-mail e senha fornecidos pelo administrador.
5. Confirme que o painel exibe os módulos permitidos para o perfil. Usuários comuns não veem controles de atualização do servidor.

## Atualizações futuras

Ao abrir, o aplicativo consulta o catálogo oficial. Quando aparecer **Atualização disponível — instalar agora?**:

1. Salve o trabalho e feche rotinas secundárias.
2. Clique em **Instalar agora**.
3. Aguarde download, verificação da assinatura, instalação e reinício.
4. Não desligue o computador durante a instalação.

O botão **Depois** é permitido apenas quando a versão instalada ainda é compatível. Se o backend exigir uma versão mínima, a atualização se torna obrigatória.

## Validação após instalar

- o app abre sem erro de compatibilidade;
- o login chega ao domínio `api.venturerp.com` por HTTPS;
- o painel abre as rotinas do perfil;
- a versão instalada corresponde à release baixada;
- uma checagem de update não apresenta erro de assinatura.

## Solução de problemas

- **Não foi possível validar o servidor:** confira internet, data/hora do Windows, proxy/firewall e acesso HTTPS ao domínio da API.
- **WebView2 ausente:** instale o Microsoft Edge WebView2 Runtime oficial e reinicie o computador.
- **Erro de assinatura:** não tente contornar. Remova o download e obtenha a release oficial novamente; informe versão e horário ao suporte.
- **Instalação bloqueada pela empresa:** peça à TI para distribuir o MSI oficial.
- **Login recusado:** confirme credencial e perfil com o administrador; reinstalar não redefine senha.
- **Atualização interrompida:** reabra o app. O pacote só é aplicado após verificação; tente novamente com conexão estável.

Ao solicitar suporte, envie versão do Windows, versão do ERP Venture, horário, texto integral do erro e uma captura sem senhas ou dados sensíveis.
