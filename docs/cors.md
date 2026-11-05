# CORS e URL da API

O backend usa `github.com/go-chi/cors`. A lista fixa anterior não incluía
`http://localhost:5580`; por isso o preflight retornava 200 sem autorização CORS.
`http://201.23.83.151` já estava autorizado no código local. Uma falha nessa
origem exige conferir a imagem implantada e a origem/URL exibidas no navegador.

Agora `CORS_ALLOWED_ORIGINS` configura a lista em tempo de execução, separada por
vírgulas. Espaços e entradas vazias são ignorados. Uma lista não vazia substitui
os padrões; variável ausente, vazia ou composta só de entradas vazias usa as
quatro origens de `.env.example`. Use origens exatas (protocolo, host e porta,
sem caminho ou barra final). Não use `*`.

No serviço `app` do Docker Compose:

```yaml
environment:
  CORS_ALLOWED_ORIGINS: ${CORS_ALLOWED_ORIGINS:-http://localhost:3000,http://127.0.0.1:3000,http://localhost:5580,http://201.23.83.151}
```

O Compose local já contém essa configuração. Adicione a variável ao `.env` do
Compose se quiser alterar a lista. Um `.env` usado apenas para interpolação pelo
Compose não injeta variáveis automaticamente; mantenha a entrada `environment`.
Os métodos GET, POST, PUT, DELETE e OPTIONS e cabeçalhos Accept, Authorization e
Content-Type são preservados. O middleware global trata preflight antes da
entrada nas rotas e acrescenta CORS também às respostas de erro.

## Imagens e frontend

Esta primeira correção exige nova imagem do backend: a imagem antiga não lê
`CORS_ALLOWED_ORIGINS`. Para o Compose local com `build: .`, quando decidir aplicar:

```sh
docker compose build app
docker compose up -d --no-deps app
```

Em produção, gere/publique uma nova tag da imagem corrigida, atualize `image:`
no serviço `app` e então execute `docker compose pull app` e
`docker compose up -d --no-deps app`. A tag antiga `1.0.0` não recebe esta correção
por adicionar somente a variável. Após essa primeira atualização, mudanças na
lista exigem apenas recriar o container (`docker compose up -d --no-deps
--force-recreate app`); `docker compose restart` não atualiza o ambiente.
Nenhum deploy é realizado por esta alteração.

O frontend usa Axios com `import.meta.env.VITE_API_URL`, padrão
`/api`, e envia o JWT no cabeçalho Authorization.
O acesso integrado usa o proxy Nginx sem IP no build (veja [proxy reverso](proxy-reverso.md)).
Somente para acesso direto a uma API externa, é possível sobrescrever o padrão:

```sh
docker build --build-arg VITE_API_URL=http://201.23.83.151:8080 -t SUA_IMAGEM_WEB:NOVA_TAG ../gastano-menos-web
```

Ou configure `build.args.VITE_API_URL` no serviço frontend do Compose.
`VITE_API_URL` é incorporada no build Vite: defini-la apenas em `environment`
no container nginx não altera os arquivos já gerados. Confira a URL da requisição
na aba Network do navegador antes de decidir reconstruir o frontend.

## Validação

Na pasta do backend, execute `go test ./...`. Os testes de CORS verificam as
quatro origens padrão, substituição por ambiente, espaços/entradas vazias,
origem não permitida, preflight de login e outra rota, cabeçalhos de autorização,
e o POST com JSON inválido no handler real de login (400, sem banco).

Após aplicar a imagem, execute para cada origem:

```sh
for origin in http://localhost:5580 http://201.23.83.151; do
  curl -i -X OPTIONS http://localhost:8080/auth/login \
    -H "Origin: $origin" \
    -H 'Access-Control-Request-Method: POST' \
    -H 'Access-Control-Request-Headers: content-type,authorization'
done
```

Espere 200, `Access-Control-Allow-Origin` igual à origem enviada,
`Access-Control-Allow-Methods: POST`, headers Content-Type e Authorization em
`Access-Control-Allow-Headers`, e `Access-Control-Allow-Credentials: true`.

```sh
curl -i http://localhost:8080/auth/login \
  -H 'Origin: http://localhost:5580' -H 'Content-Type: application/json' --data '{'
curl -i http://localhost:8080/categories -H 'Origin: http://localhost:5580'
```

Espere 400 e 401 respectivamente, ambos com `Access-Control-Allow-Origin`.
Para validar login bem-sucedido, use a interface com uma conta existente e confira
200 e token na resposta; não registre nem compartilhe o token. Confira também
as consultas autenticadas no dashboard. Os testes locais não confirmam a imagem
atual no servidor, conectividade pública, credenciais reais ou login contra o
PostgreSQL de produção. Nenhum dado ou regra de autenticação foi alterado.
