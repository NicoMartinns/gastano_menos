# Administração de acesso e ambiente compartilhado

As contas controlam a entrada em um único ambiente financeiro. Todos os usuários
liberados podem consultar, criar, editar e excluir as mesmas transações e categorias.
Os relatórios consideram todos os lançamentos. O `user_id` continua identificando
quem criou cada registro; ele não limita o acesso e não é sobrescrito na edição.
Não há separação por organização ou por usuário.

## Perfis e liberação

- `USER`: acesso às funções financeiras quando `is_active = true`.
- `ADMIN`: mesmas funções e gestão de contas em `/admin/users` no frontend.
- Conta bloqueada: não pode entrar nem usar um JWT anterior ao bloqueio.

O backend consulta a conta a cada requisição autenticada. Mudanças de perfil ou
liberação incrementam `session_version`, invalidando os tokens anteriores mesmo
se a conta for liberada novamente. A interface revalida a sessão ao ganhar foco e
a cada minuto; qualquer requisição com sessão revogada recebe 401 imediatamente.
Um usuário comum recebe 403 nas rotas administrativas, mesmo acessando a API diretamente.

Administradores podem cadastrar contas, editar nome/email/perfil e liberar ou
bloquear acesso. Novas contas na tela começam com a liberação desmarcada.
Não é permitido bloquear ou rebaixar a própria conta nem remover o último
administrador ativo. A verificação do último administrador e a atualização usam
uma transação com lock compartilhado pelas operações administrativas, inclusive
entre instâncias da API.

Não há cadastro público, exclusão de contas ou redefinição de senha nesta etapa.
O bloqueio preserva os registros financeiros criados pela conta.

## Migration preparada, não executada

`000006_add_user_access.up.sql` adiciona `role`, `is_active` e `session_version`.
Os usuários existentes permanecem como `USER`, liberados, sem trocar suas senhas.
Tokens antigos continuam válidos até expirar se a conta permanecer liberada e
sem alteração de perfil/liberação. Não é criado um administrador automaticamente.

A API já aplica migrations ao iniciar. Portanto, iniciar a nova versão contra um
banco existente executará também a migration 000006. A publicação precisa ser
planejada com backup e validação prévia em ambiente descartável. Este trabalho
não iniciou a API nem executou migrations em banco algum.

O rollback remove os campos de acesso, perdendo essa configuração. Não use a
versão nova da aplicação sobre o schema anterior.

## Criar o primeiro administrador

Depois que a migration estiver aplicada no ambiente escolhido, use o comando
explícito `cmd/create-admin`. Ele **não executa migrations**, não altera contas
existentes e se recusa a criar o administrador inicial se já existir qualquer
conta com perfil `ADMIN`. Escolha um email ainda não cadastrado. Depois, os demais
administradores podem ser cadastrados ou promovidos pela interface.

As variáveis `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` e `DB_NAME` devem apontar
para o banco desejado, como na API. O comando também aceita a configuração local
via `.env`. Não coloque a senha no código nem como argumento de linha de comando.

Exemplo em **zsh**, na raiz do backend (substitua nome e email):

```zsh
read -rs 'ADMIN_PASSWORD?Senha inicial (12 a 72 bytes): '
printf '\n'
export ADMIN_PASSWORD
go run ./cmd/create-admin --name 'Administrador' --email 'admin@seu-dominio.com'
unset ADMIN_PASSWORD
```

O Dockerfile também inclui o binário `/app/create-admin`. Com a variável
`ADMIN_PASSWORD` exportada no shell, execute explicitamente no container alvo:

```zsh
docker exec -e ADMIN_PASSWORD gastano-menos ./create-admin \
  --name 'Administrador' --email 'admin@seu-dominio.com'
unset ADMIN_PASSWORD
```

O nome do container deve corresponder ao ambiente. Este comando grava uma conta;
ele é uma instrução para execução posterior, não foi executado nesta implementação.

`JWT_SECRET` deve ser configurado e não vazio. A API interrompe a inicialização
antes de abrir o banco quando essa configuração não existe.

## API

Todos os endpoints abaixo exigem `Authorization: Bearer <token>`.

| Método | Caminho | Permissão | Conteúdo |
| --- | --- | --- | --- |
| GET | `/auth/me` | Conta liberada | Identidade, perfil e liberação atuais |
| GET | `/admin/users/?limit=20&offset=0` | ADMIN | `{ "users": [...], "total": 0 }`; limite de 1 a 100 |
| POST | `/admin/users/` | ADMIN | Cadastro com nome, email, senha, perfil e liberação |
| PUT | `/admin/users/{id}` | ADMIN | Atualização de nome, email, perfil e liberação |

Cadastro:

```json
{
  "name": "Nome do usuário",
  "email": "usuario@seu-dominio.com",
  "password": "senha-definida-no-ambiente",
  "role": "USER",
  "is_active": true
}
```

A senha deve ter entre 12 e 72 bytes e é armazenada como hash bcrypt. No PUT,
envie os mesmos campos, exceto `password`; alteração de senha não é suportada.
As respostas nunca incluem hash de senha ou versão interna de sessão.

Erros previstos: 400 para payload inválido; 401 para sessão inválida/bloqueada;
403 para falta de perfil; 404 para conta inexistente; 409 para email duplicado ou
mudança proibida de administrador. Erros internos não são retornados ao cliente.

## Frontend e implantação

O frontend consulta `/auth/me` para proteger as telas e mostrar a opção **Usuários**
no menu desktop/mobile de administradores. Esconder essa opção não substitui a
autorização do backend.

O frontend usa `/api` por padrão; o Nginx encaminha para `app:8080` na rede
Compose, sem IP público no bundle. Consulte [proxy reverso](proxy-reverso.md)
para operação e rollback. `VITE_API_URL` é uma sobrescrita pública de **build**
para API externa; nesse caso, autorize a origem no CORS. Não inclua segredos.
Nenhum deploy foi executado.

## Validação sem banco

Os testes usam stubs e um driver SQL de teste, sem conexões reais:

```sh
go test ./...
go vet ./...
```

Eles cobrem login, bloqueio, expiração/assinatura de JWT, revogação após nova
liberação, autorização administrativa, validação/cadastro de contas, proteção do
último administrador, sigilo nas respostas e consultas financeiras compartilhadas.
Ainda é necessário validar a migration e os fluxos completos em um banco de QA
antes de publicar. Não execute os testes de implantação em produção.

## Restauração local de QA — 01/11/2026

O dump produtivo foi restaurado no PostgreSQL local (`gastano_postgres`),
exclusivamente em `gastano_db_qa`. A API `gastano_app` foi parada durante a
restauração e reiniciada depois, aplicando a migration 000006 automaticamente.
Validação: versão 6 sem estado dirty, 1 usuário, 64 categorias e 1.164 transações.
O usuário importado permanece ativo com perfil `USER`; o dump não contém ADMIN.
Para testar a administração, use o procedimento de criação do primeiro
administrador acima, apontando explicitamente para QA.

O `dump.sql` original foi preservado. Foram encontrados 29 separadores de campos
TAB ausentes: 27 entre tipo e data de criação nas categorias e 2 entre dia de
recorrência e origem nula nas transações. A cópia corrigida usada na importação
está em `backups/dump-qa-corrigido-20261101.sql`.

O backup anterior do QA está em `backups/qa-before-restore-20261101.dump`
(formato custom do pg_dump). Dumps, backups e arquivos de ambiente estão
excluídos do contexto Docker; dumps e backups também estão ignorados no Git.

Para recuperar o estado anterior do QA, pare a API, restaure e reinicie:

```sh
docker stop gastano_app
docker exec -i gastano_postgres pg_restore -U gastano_admin -d gastano_db_qa \
  --clean --if-exists --single-transaction --exit-on-error \
  < backups/qa-before-restore-20261101.dump
docker start gastano_app
```

A restauração de QA não executa build, push ou deploy em produção.
