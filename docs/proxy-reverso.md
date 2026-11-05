# Proxy reverso frontend/backend

Antes, a URL padrão `http://localhost:8080` apontava para o computador do usuário,
e uma URL pública definida em VITE_API_URL ficava incorporada ao bundle.
Agora o Axios usa `/api` e o Nginx do container web encaminha para `app:8080`.
A barra final de `proxy_pass http://app:8080/` remove `/api/`, preservando o
restante do caminho, query string, método e corpo. O fallback React permanece
em `location /`; erros da API não são convertidos em index.html.
Authorization e Cookie são encaminhados por padrão pelo Nginx, assim como as
respostas e Set-Cookie do backend. A autenticação atual usa Bearer JWT e não
foi alterada. O CORS do backend permanece para clientes externos; o navegador
não exige CORS para chamadas relativas na mesma origem.

## Compose e imagens

O Compose do backend agora inclui web, app e postgres na rede padrão do mesmo
projeto. Não é necessário criar rede adicional. Os repositórios devem continuar
lado a lado para o contexto `../gastano-menos-web` funcionar no build local.
As publicações 8080 e 5432 ficam em 127.0.0.1; apenas 80 permanece pública.
O Nginx acessa app:8080 pela rede interna, independentemente dessa publicação.
Os volumes, credenciais e dados do PostgreSQL não foram alterados.

É necessária uma nova imagem **frontend** (bundle e nginx.conf). Esta mudança
não exige nova imagem backend. Construa sem URL pública, com VITE_API_URL vazio
ou `/api`. Após isso, trocar IP/domínio ou acessar por túnel não exige rebuild.

Para servidor com imagens prontas, mantenha o projeto Compose, serviço postgres,
volume e configurações existentes. No serviço web, use a nova tag publicada:

```yaml
web:
  image: nicolasmartinss/gastano-menos-web:NOVA_TAG
  ports:
    - "80:80"
  depends_on:
    - app
```

Mantenha web e app na mesma rede Compose (a padrão basta). Se houver redes
explícitas, conecte ambos à mesma. Em app e postgres, substitua respectivamente
as publicações por `127.0.0.1:8080:8080` e `127.0.0.1:5432:5432`, se precisar
manter acesso de manutenção local/SSH. Elas também podem ser removidas se não
houver consumidores externos; antes confira integrações que usem essas portas.
Não renomeie o projeto ou o volume existente nem execute `down -v`.

Quando decidir publicar, guarde a tag anterior e uma cópia do Compose, gere e
publique uma nova tag web. No servidor, execute apenas:

```sh
docker compose config --quiet
docker compose pull web
docker compose up -d --no-deps web
```

Esses comandos atualizam somente web. Alterações de portas em app/postgres
exigem recriar esses serviços em janela de manutenção separada; isso não é
necessário para o proxy funcionar. No ambiente local com build disponível:
`docker compose build web` e `docker compose up -d --no-deps web`.
Nenhum desses comandos foi executado automaticamente nesta alteração.
Se app for recriado com outro IP interno, recarregue/recrie web também, pois
o upstream estático é resolvido quando o Nginx inicia.

## Validação após publicação

```sh
docker compose exec web nginx -t
curl -i http://localhost/
curl -i http://localhost/admin/users
curl -i http://localhost/api/categories
curl -i http://localhost/api/auth/login \
  -H 'Content-Type: application/json' --data '{'
curl -i 'http://localhost/api/transactions?month=2026-11'
```

Espere HTML 200 nos dois primeiros casos; 401 nas consultas sem token e 400
no login com JSON inválido. Os erros de API devem vir do Go, sem HTML React.
Para confirmar remoção do prefixo/query, confira a requisição no backend em um
ambiente de teste; `/api/auth/login` precisa chegar como `/auth/login`.

Abra `http://IP_DO_SERVIDOR` e teste login com conta existente, dashboard e
administração conforme seu perfil; atualize uma página interna. Confira na aba
Network que as chamadas usam `/api` na mesma origem, recebem token no login e
enviam Authorization nas consultas. Não compartilhe tokens. Teste também por:

```sh
ssh -N -L 5580:localhost:80 USUARIO@IP_DO_SERVIDOR
```

Acesse `http://localhost:5580` e repita. A URL da API deve usar a porta 5580,
sem acessar o IP público ou localhost:8080. Cookies que clientes futuros
adicionarem precisam ter Domain/Path/Secure compatíveis com a URL pública;
o proxy não reescreve esses atributos.

## Rollback

Restaure a tag anterior de web e sua configuração anterior de VITE_API_URL,
execute `docker compose pull web` e `docker compose up -d --no-deps web`.
Se a imagem anterior acessa a porta 8080 diretamente, restaure também sua
publicação anterior e a allowlist CORS necessária antes de testar. Não reverta
banco/migrations nem remova volumes. Se mudou redes/portas, restaure a cópia do
Compose e recrie somente os serviços afetados em janela apropriada.
