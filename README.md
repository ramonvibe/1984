# 1984

Gerenciador de tarefas, projetos e sprints para pequenas equipes de software. A aplicação é um monólito em Go, usa PostgreSQL e renderiza HTML no servidor. Não precisa de Node.js, Redis ou processo separado de frontend.

> Status: MVP. O fluxo principal funciona, mas o projeto ainda está em desenvolvimento e não foi consolidado para produção.

## Rodar pela primeira vez

O caminho recomendado usa Docker. Você precisa apenas de:

- [Git](https://git-scm.com/downloads)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) no Windows ou macOS; no Linux, Docker Engine com o plugin Compose

Clone e inicie:

```bash
git clone https://github.com/ramonvibe/1984.git
cd 1984
docker compose up --build
```

Quando aparecer `1984 is ready`, abra [http://localhost:8080](http://localhost:8080). No primeiro acesso, a aplicação pedirá:

1. nome do espaço de trabalho;
2. seu nome;
3. email;
4. uma senha com pelo menos 12 caracteres.

Essa primeira conta será administradora. As tabelas e atualizações do banco são aplicadas automaticamente ao iniciar.

Para parar, pressione `Ctrl+C`. Seus dados continuam salvos no volume do PostgreSQL. Nas próximas vezes, execute:

```bash
docker compose up
```

## Comandos úteis

```bash
# Iniciar em segundo plano
docker compose up -d

# Ver os logs
docker compose logs -f app

# Parar
docker compose down

# Atualizar depois de um git pull
docker compose up --build -d
```

Para apagar completamente o banco local e começar novamente:

```bash
docker compose down -v
```

Esse último comando remove permanentemente usuários, projetos, tarefas e todo o restante armazenado no ambiente local.

### Porta já ocupada

Se a porta `8080` estiver em uso, escolha outra antes de iniciar:

```bash
PORT=8081 docker compose up --build
```

Depois acesse `http://localhost:8081`. Se a porta `5432` do PostgreSQL estiver ocupada, use:

```bash
POSTGRES_PORT=5433 docker compose up --build
```

## Desenvolvimento sem colocar a aplicação no Docker

Requisitos: Go 1.25 ou superior, Git e Bash. O Docker pode executar somente o PostgreSQL:

```bash
git clone https://github.com/ramonvibe/1984.git
cd 1984
docker compose up -d db
cp .env.example .env
make dev
```

Sem Make, use `bash scripts/dev.sh`. Acesse [http://localhost:8080](http://localhost:8080). Reinicie o processo após alterar código; ainda não existe recarregamento automático.

O arquivo `.env` é ignorado pelo Git. `scripts/dev.sh` carrega esse arquivo como Bash, portanto use apenas conteúdo confiável.

### Testes e arquivos gerados

```bash
go test ./...
```

Templates e consultas geradas já estão versionados. Só é necessário regenerá-los ao editar seus arquivos de origem:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

## Funcionalidades

- Interface em português brasileiro, com modo claro e escuro.
- Espaço de trabalho com administradores e membros.
- Projetos com códigos legíveis, como `PLAT-142`.
- Tarefas, erros, funcionalidades e melhorias com prioridade, etiquetas, responsável e histórico.
- Quadro com arrastar e soltar.
- Preparação de sprint: selecione uma sprint, escolha tarefas movendo-as para **A fazer** e inicie a sprint.
- Comentários em Markdown e vínculo opcional com commits e pull requests do GitHub.
- Cronômetro, registros manuais e relatórios de tempo.
- Calendário, busca, versões e geração de histórico de alterações.

## Configuração

A instalação via Docker já possui valores locais para começar. Para outra instalação, configure estas variáveis no ambiente:

| Variável | Finalidade | Padrão |
| --- | --- | --- |
| `DATABASE_URL` | URL de conexão com PostgreSQL | banco local `trackline` |
| `ADDR` | endereço HTTP de escuta | `:8080` |
| `BASE_URL` | URL pública da aplicação | `http://localhost:8080` |
| `SECURE_COOKIES` | força cookies seguros | ativo quando `BASE_URL` usa HTTPS |
| `GITHUB_APP_ID` | ID opcional do GitHub App | vazio |
| `GITHUB_APP_SLUG` | identificador do GitHub App | vazio |
| `GITHUB_PRIVATE_KEY` | chave RSA do GitHub App em PEM | vazio |
| `GITHUB_WEBHOOK_SECRET` | segredo do webhook, mínimo de 32 bytes | vazio |
| `APP_VERSION` | versão instalada usada na comparação | `0.1.0` |
| `UPDATE_CHECK_URL` | rota da última release no GitHub | repositório oficial do 1984 |

A integração com GitHub é opcional. Sem essas quatro variáveis, todo o restante funciona normalmente.

O 1984 consulta a release mais recente do repositório uma vez ao iniciar e depois a cada 24 horas. Quando a tag publicada for superior a `APP_VERSION`, aparece um aviso no topo. Ao publicar uma versão, use tags semânticas como `v0.2.0` e atualize `APP_VERSION` na instalação.

## Publicação

O `Dockerfile` gera um único executável com templates, arquivos estáticos e migrações incorporados. Para publicar:

- use uma senha forte e exclusiva para o PostgreSQL;
- defina `BASE_URL` com a URL HTTPS pública;
- coloque a aplicação atrás de um proxy reverso com TLS;
- não exponha a porta do PostgreSQL à internet;
- mantenha backups do volume do banco;
- conclua o primeiro cadastro antes de liberar uma instalação vazia publicamente.

O endpoint `GET /healthz` verifica a conexão com o banco. As credenciais do `compose.yaml` destinam-se somente ao desenvolvimento local.

## Organização

```text
cmd/server/       entrada da aplicação
db/migrations/    alterações versionadas do banco
db/queries/       consultas usadas pelo sqlc
internal/         regras de negócio, banco e HTTP
web/              templates, CSS e JavaScript
```

## Limitações atuais

- Membros de projetos servem para organização; todos os membros do espaço de trabalho acessam todos os projetos.
- A integração com GitHub exige criar e configurar um GitHub App manualmente.
- Ainda não há procedimento automatizado de backup e restauração.
- Ainda não foi escolhida uma licença. Não presuma direitos de redistribuição até a inclusão de uma licença.
