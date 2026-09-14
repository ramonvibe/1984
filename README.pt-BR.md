# 1984

[English](README.md) · [Português brasileiro](README.pt-BR.md)

Um gerenciador leve de issues e projetos, auto-hospedável, para pequenas equipes de desenvolvimento.

> Tudo que uma pequena equipe de software precisa. Nada que ela não precise.

1984 é um monólito modular em Go com PostgreSQL e HTML renderizado no servidor. Não exige Node.js, etapa de build de frontend, Redis nem worker separado para funcionar.

Esta é uma versão funcional inicial. A interface e o módulo Go ainda utilizam o nome provisório **Trackline**.

## Funcionalidades

- Criação inicial de workspace; o primeiro usuário vira administrador. Administradores podem cadastrar membros.
- Projetos com identificadores legíveis de issues, como `PLAT-142`.
- Bugs, funcionalidades, melhorias e tarefas; status e prioridades predefinidos.
- Edição de propriedades com HTMX, etiquetas, comentários em Markdown e histórico de atividades.
- Board com arrastar e soltar; cards mostram o primeiro nome do responsável ao lado do avatar.
- Sprints com datas de início/fim, duração incluindo ambas as datas e filtro no board. Vincule issues pelo campo Sprint.
- Um cronômetro ativo por usuário, registro manual de tempo e comparação entre estimado e realizado.
- Dashboard e relatórios de tempo por membro, projeto, issue, tipo e release.
- Calendário com datas das issues e metas das releases; busca usando PostgreSQL.
- Releases com rascunhos editáveis de changelog, gerados deterministicamente a partir das issues concluídas.
- Integração opcional via GitHub App para vincular commits/PRs e publicar GitHub Releases.

## Executar localmente

Requisitos: Go 1.25 ou superior, PostgreSQL (testado na versão 18), Git, Bash e, opcionalmente, Make.

```bash
git clone https://github.com/ramonvibe/1984.git
cd 1984
```

Crie um banco dedicado e um usuário sem privilégios de superusuário usando uma conta administradora do PostgreSQL. Adapte as opções de conexão à sua instalação:

```bash
psql -h 127.0.0.1 -U postgres -d postgres
```

Dentro do `psql`:

```sql
CREATE ROLE trackline LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
\password trackline
CREATE DATABASE trackline OWNER trackline;
REVOKE ALL ON DATABASE trackline FROM PUBLIC;
\q
```

Se outro projeto já utiliza PostgreSQL, reutilize o servidor, mas mantenha este banco e usuário separados. Não sobrescreva um banco existente com o mesmo nome.

Crie `.env` na raiz do repositório, substituindo a senha do exemplo pela escolhida. Codifique caracteres especiais da senha no formato de URL:

```dotenv
DATABASE_URL='postgres://trackline:SUA_SENHA@127.0.0.1:5432/trackline?sslmode=disable'
ADDR=':8080'
BASE_URL='http://localhost:8080'
```

```bash
chmod 600 .env
make dev
# Sem Make: bash scripts/dev.sh
```

Acesse [localhost:8080](http://localhost:8080), crie seu workspace e a conta administradora, depois crie um projeto. O servidor aplica migrations pendentes automaticamente. Reinicie `make dev` após alterações no código; não há recarregamento automático.

O Git ignora `.env`. O script de desenvolvimento interpreta esse arquivo como Bash, então utilize apenas um arquivo confiável. `sslmode=disable` serve para desenvolvimento local, não para redes não confiáveis.

## Configuração e hospedagem

A aplicação lê variáveis de ambiente. Somente o script de desenvolvimento carrega `.env` automaticamente.

| Variável | Finalidade |
| --- | --- |
| `DATABASE_URL` | URL de conexão PostgreSQL; configure explicitamente para sua instalação. |
| `ADDR` | Endereço de escuta HTTP; padrão `:8080`. |
| `BASE_URL` | URL pública da aplicação; padrão `http://localhost:8080`. |
| `SECURE_COOKIES` | Por padrão, fica true quando `BASE_URL` começa com `https://`. |
| `GITHUB_APP_ID` | ID opcional da GitHub App. |
| `GITHUB_APP_SLUG` | Slug opcional da App usado no link de instalação. |
| `GITHUB_PRIVATE_KEY` | Chave privada RSA da App em PEM; aceita sequências literais `\n`. |
| `GITHUB_WEBHOOK_SECRET` | Segredo do webhook da App, com pelo menos 32 bytes. |

Sem configuração GitHub, o tracker funciona de forma independente. Após configurar e instalar a App, conecte o ID da instalação e o repositório nas configurações do projeto. O endpoint de webhook é `POST /webhooks/github`. Mantenha chaves e segredos fora do Git.

Compile um único executável com templates, arquivos estáticos e migrations incorporados:

```bash
go build -o /tmp/1984-server ./cmd/server
# Com DATABASE_URL e demais variáveis exportadas:
/tmp/1984-server -migrate
/tmp/1984-server
```

Para hospedar, utilize HTTPS por um proxy reverso, configure a `BASE_URL` pública, proteja o acesso ao banco e mantenha backups do PostgreSQL. Conclua o cadastro inicial antes de expor uma instalação vazia publicamente. `GET /healthz` verifica a conexão com o banco. Revise segurança e procedimentos de recuperação antes do uso em produção.

## Desenvolvimento

```bash
go test ./...
```

O teste opcional de integração de sprint com o banco é executado quando `TEST_DATABASE_URL` está definida. Use um banco de testes cujo usuário possa criar schemas; o teste cria e remove um schema temporário isolado.

Templates e código das consultas SQL já estão versionados; os geradores não são necessários para uma compilação normal. Após editar arquivos `.templ` ou consultas SQL, regenere os arquivos Go correspondentes:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

Mantenha handlers HTTP em `internal/server`, regras de negócio nos pacotes de domínio, SQL em `db/queries`, alterações de schema em novos arquivos de `db/migrations` e apresentação em `web`. Não edite manualmente os arquivos gerados `*_templ.go` e de consultas, nem altere migrations já aplicadas a um banco.

Contribuições devem ser pequenas, incluir testes relevantes e evitar dependências desnecessárias. Nunca versione `.env`, credenciais ou dumps do banco.

## Limitações atuais

- A interface está somente em inglês; a tradução para português está pendente. O README está disponível nos dois idiomas.
- O vínculo de sprint é manual. A associação automática de issues a partir de Todo à sprint atual está pendente.
- Dockerfile e Docker Compose ainda não estão incluídos.
- Membros de projetos servem para organização: todos os membros do workspace acessam todos os projetos. Não há controle de acesso por projeto.
- Ainda não foi escolhida uma licença. Não presuma direitos de redistribuição open source até a inclusão de uma licença.
