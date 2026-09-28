package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramon/trackline/internal/auth"
	"github.com/ramon/trackline/internal/comment"
	"github.com/ramon/trackline/internal/config"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/internal/github"
	"github.com/ramon/trackline/internal/issue"
	"github.com/ramon/trackline/internal/project"
	"github.com/ramon/trackline/internal/release"
	"github.com/ramon/trackline/internal/timetracking"
	"github.com/ramon/trackline/web/layouts"
	"github.com/ramon/trackline/web/pages"
	"github.com/ramon/trackline/web/static"
)

type App struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	Queries  *database.Queries
	Auth     auth.Service
	Projects project.Service
	Issues   issue.Service
	Comments comment.Service
	Time     timetracking.Service
	Releases release.Service
	GitHub   github.Service
	loginMu  sync.Mutex
	attempts map[string]attempt
}
type attempt struct {
	Count int
	Until time.Time
}
type contextKey int

const userKey contextKey = 0
const csrfKey contextKey = 1

type handler func(http.ResponseWriter, *http.Request) error

func New(pool *pgxpool.Pool, configuration config.Config) (*App, error) {
	client, err := github.NewClient(configuration.GitHubAppID, configuration.GitHubPrivateKey, configuration.GitHubWebhookSecret, configuration.GitHubAPIURL)
	if err != nil {
		return nil, err
	}
	q := database.New(pool)
	return &App{Config: configuration, Pool: pool, Queries: q, Auth: auth.Service{Pool: pool, Queries: q}, Projects: project.Service{Pool: pool, Queries: q}, Issues: issue.Service{Pool: pool, Queries: q}, Comments: comment.Service{Pool: pool}, Time: timetracking.Service{Pool: pool}, Releases: release.Service{Pool: pool, Queries: q, GitHub: client}, GitHub: github.Service{Pool: pool, Queries: q, Client: client}, attempts: map[string]attempt{}}, nil
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	public := func(pattern string, h handler) { mux.HandleFunc(pattern, a.wrap(h, false)) }
	private := func(pattern string, h handler) { mux.HandleFunc(pattern, a.wrap(h, true)) }
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static.Files)))
	public("GET /healthz", func(w http.ResponseWriter, r *http.Request) error {
		if err := a.Pool.Ping(r.Context()); err != nil {
			return err
		}
		_, err := io.WriteString(w, "ok\n")
		return err
	})
	public("GET /setup", a.authPage)
	public("POST /setup", a.setup)
	public("GET /login", a.authPage)
	public("POST /login", a.login)
	public("GET /invite/{token}", a.invitePage)
	public("POST /invite/{token}", a.acceptInvite)
	public("GET /brand/logo", a.brandLogo)
	// GitHub authenticates by HMAC rather than browser sessions or CSRF tokens.
	mux.HandleFunc("POST /webhooks/github", a.webhook)
	private("POST /logout", a.logout)
	private("GET /{$}", a.dashboard)
	private("GET /projects", a.projects)
	private("GET /my-tasks", a.myTasks)
	private("POST /projects", a.createProject)
	private("GET /projects/{key}", a.projectPage)
	private("GET /projects/{key}/{tab}", a.projectPage)
	private("GET /projects/{key}/issues/new", a.newIssue)
	private("POST /projects/{key}/issues", a.createIssue)
	private("POST /projects/{key}/sprints", a.createSprint)
	private("POST /projects/{key}/sprints/start", a.startSprint)
	private("POST /projects/{key}/sprints/select", a.selectSprint)
	private("POST /issues/{ref}/sprint", a.setIssueSprint)
	private("POST /projects/{key}/settings", a.updateProject)
	private("POST /projects/{key}/members", a.projectMember)
	private("POST /projects/{key}/github", a.connectGitHub)
	private("POST /projects/{key}/releases", a.createRelease)
	private("GET /issues/{ref}", a.issuePage)
	private("POST /issues/{ref}", a.updateIssue)
	private("POST /issues/{ref}/status", a.moveIssue)
	private("POST /issues/{ref}/comments", a.createComment)
	private("POST /issues/{ref}/time", a.trackTime)
	private("POST /issues/{ref}/subtasks", a.addSubtask)
	private("POST /issues/{ref}/subtasks/{id}", a.toggleSubtask)
	private("GET /releases/{id}", a.releasePage)
	private("POST /releases/{id}", a.updateRelease)
	private("POST /releases/{id}/generate", a.generateChangelog)
	private("POST /releases/{id}/draft", a.saveChangelog)
	private("POST /releases/{id}/publish", a.publishRelease)
	private("GET /calendar", a.calendarPage)
	private("GET /time", a.timePage)
	private("GET /search", a.search)
	private("GET /settings", a.settings)
	private("GET /profile/avatar", a.profileAvatar)
	private("POST /settings/profile", a.updateProfile)
	private("POST /settings/users", a.addUser)
	private("POST /settings/labels", a.addLabel)
	private("POST /settings/branding", a.updateBranding)
	private("POST /settings/sprint-options", a.updateSprintOptions)
	private("POST /settings/invites", a.createInvite)
	private("POST /settings/users/{id}/role", a.updateUserRole)
	private("POST /workspaces/{id}/switch", a.switchWorkspace)
	origin := http.NewCrossOriginProtection()
	return a.headers(origin.Handler(mux))
}
func (a *App) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		defer func() {
			if failure := recover(); failure != nil {
				slog.Error("request panic", "path", r.URL.Path, "error", failure)
				http.Error(w, "Erro interno do servidor", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func (a *App) cookieName(name string) string {
	if a.Config.SecureCookies {
		return "__Host-trackline_" + name
	}
	return "trackline_" + name
}
func (a *App) cookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: a.cookieName(name), Value: value, Path: "/", HttpOnly: true, Secure: a.Config.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func (a *App) wrap(h handler, private bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		csrf := ""
		if cookie, err := r.Cookie(a.cookieName("csrf")); err == nil && len(cookie.Value) == 43 {
			csrf = cookie.Value
		}
		if csrf == "" {
			csrf = auth.Token()
			a.cookie(w, "csrf", csrf, 30*86400)
		}
		r = r.WithContext(context.WithValue(r.Context(), csrfKey, csrf))
		r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
		if r.Method != "GET" && r.Method != "HEAD" {
			var err error
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				err = r.ParseMultipartForm(6 << 20)
			} else {
				err = r.ParseForm()
			}
			if err != nil {
				http.Error(w, "Formulário inválido ou grande demais", 400)
				return
			}
			supplied := r.Header.Get("X-CSRF-Token")
			if supplied == "" {
				supplied = r.PostForm.Get("csrf")
			}
			if subtle.ConstantTimeCompare([]byte(supplied), []byte(csrf)) != 1 {
				http.Error(w, "Token de segurança inválido. Recarregue a página e tente novamente.", 403)
				return
			}
		}
		if private {
			cookie, err := r.Cookie(a.cookieName("session"))
			if err != nil {
				a.loginRedirect(w, r)
				return
			}
			user, err := a.Queries.GetSessionUser(r.Context(), auth.TokenHash(cookie.Value))
			if errors.Is(err, pgx.ErrNoRows) {
				a.loginRedirect(w, r)
				return
			}
			if err != nil {
				a.fail(w, r, err)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), userKey, user))
		}
		if err := h(w, r); err != nil {
			a.fail(w, r, err)
		}
	}
}
func (a *App) loginRedirect(w http.ResponseWriter, r *http.Request) {
	count, err := a.Queries.WorkspaceCount(r.Context())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	target := "/login"
	if count == 0 {
		target = "/setup"
	}
	redirect(w, r, target)
}
func User(r *http.Request) auth.User { user, _ := r.Context().Value(userKey).(auth.User); return user }
func csrf(r *http.Request) string    { value, _ := r.Context().Value(csrfKey).(string); return value }
func (a *App) base(r *http.Request) (layouts.Data, error) {
	data := layouts.Data{User: User(r), CSRF: csrf(r), Path: r.URL.Path, AppName: User(r).AppName, HasLogo: User(r).HasLogo}
	rows, workspacesErr := a.Pool.Query(r.Context(), "SELECT DISTINCT w.id,w.name FROM users u JOIN workspaces w ON w.id=u.workspace_id WHERE lower(u.email)=lower($1) AND u.active ORDER BY w.name", data.User.Email)
	if workspacesErr != nil {
		return data, workspacesErr
	}
	defer rows.Close()
	for rows.Next() {
		var option layouts.WorkspaceOption
		if workspacesErr = rows.Scan(&option.ID, &option.Name); workspacesErr != nil {
			return data, workspacesErr
		}
		data.Workspaces = append(data.Workspaces, option)
	}
	if workspacesErr = rows.Err(); workspacesErr != nil {
		return data, workspacesErr
	}
	var err error
	data.Projects, err = a.Queries.ListProjects(r.Context(), data.User.WorkspaceID)
	if err != nil {
		return data, err
	}
	data.Timer, err = a.Queries.GetActiveTimer(r.Context(), data.User.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return data, err
}
func render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	var buffer bytes.Buffer
	if err := component.Render(r.Context(), &buffer); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(buffer.Bytes())
	return err
}
func redirect(w http.ResponseWriter, r *http.Request, path string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}
func (a *App) fail(w http.ResponseWriter, r *http.Request, err error) {
	status := 400
	message := err.Error()
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status = 404
		message = "Não encontrado"
	case errors.Is(err, auth.ErrForbidden):
		status = 403
	case errors.As(err, &pgErr):
		if pgErr.Code == "23505" {
			status = 409
			message = "Esse nome, sigla ou registro já existe."
		} else {
			status = 500
			message = "Não foi possível salvar ou carregar os dados. Tente novamente."
		}
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		status = 503
		message = "A solicitação demorou demais. Tente novamente."
	}
	if status >= 500 {
		slog.Error("request failed", "path", r.URL.Path, "error", err)
	}
	if r.Header.Get("HX-Request") == "true" || r.Header.Get("Accept") == "application/json" {
		http.Error(w, message, status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pages.Error(layouts.Data{User: User(r), CSRF: csrf(r)}, status, message).Render(r.Context(), w)
}
func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("identificador inválido")
	}
	return id, nil
}
func (a *App) allowLogin(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	now := time.Now()
	if len(a.attempts) >= 4096 {
		for key, value := range a.attempts {
			if now.After(value.Until) {
				delete(a.attempts, key)
			}
		}
		if len(a.attempts) >= 4096 {
			return false
		}
	}
	value := a.attempts[host]
	if now.After(value.Until) {
		value = attempt{Until: now.Add(time.Minute)}
	}
	value.Count++
	a.attempts[host] = value
	return value.Count <= 10
}
func (a *App) authPage(w http.ResponseWriter, r *http.Request) error {
	count, err := a.Queries.WorkspaceCount(r.Context())
	if err != nil {
		return err
	}
	setup := count == 0
	if setup && r.URL.Path != "/setup" {
		redirect(w, r, "/setup")
		return nil
	}
	if !setup && r.URL.Path == "/setup" {
		redirect(w, r, "/login")
		return nil
	}
	appName, hasLogo := "1984", false
	if !setup {
		if err = a.Pool.QueryRow(r.Context(), "SELECT app_name, logo IS NOT NULL FROM workspaces ORDER BY id LIMIT 1").Scan(&appName, &hasLogo); err != nil {
			return err
		}
	}
	return render(w, r, pages.Auth(setup, csrf(r), "", appName, hasLogo))
}
func (a *App) setup(w http.ResponseWriter, r *http.Request) error {
	if !a.allowLogin(r) {
		http.Error(w, "Muitas tentativas. Tente novamente em um minuto.", 429)
		return nil
	}
	token, err := a.Auth.Setup(r.Context(), r.FormValue("workspace"), r.FormValue("name"), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		return err
	}
	a.cookie(w, "session", token, 30*86400)
	a.cookie(w, "csrf", auth.Token(), 30*86400)
	redirect(w, r, "/")
	return nil
}
func (a *App) login(w http.ResponseWriter, r *http.Request) error {
	if !a.allowLogin(r) {
		http.Error(w, "Muitas tentativas. Tente novamente em um minuto.", 429)
		return nil
	}
	if len(r.FormValue("password")) > 256 {
		return auth.ErrCredentials
	}
	token, err := a.Auth.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if errors.Is(err, auth.ErrCredentials) {
		w.WriteHeader(401)
		appName, hasLogo := "1984", false
		_ = a.Pool.QueryRow(r.Context(), "SELECT app_name, logo IS NOT NULL FROM workspaces ORDER BY id LIMIT 1").Scan(&appName, &hasLogo)
		return render(w, r, pages.Auth(false, csrf(r), err.Error(), appName, hasLogo))
	}
	if err != nil {
		return err
	}
	a.cookie(w, "session", token, 30*86400)
	a.cookie(w, "csrf", auth.Token(), 30*86400)
	redirect(w, r, "/")
	return nil
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie(a.cookieName("session")); err == nil {
		if err = a.Queries.DeleteSession(r.Context(), auth.TokenHash(cookie.Value)); err != nil {
			return err
		}
	}
	a.cookie(w, "session", "", -1)
	redirect(w, r, "/login")
	return nil
}
func (a *App) webhook(w http.ResponseWriter, r *http.Request) {
	if !a.GitHub.Client.Enabled() {
		http.Error(w, "GitHub não configurado", 404)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		http.Error(w, "Webhook grande demais", 413)
		return
	}
	if !github.VerifySignature(a.Config.GitHubWebhookSecret, r.Header.Get("X-Hub-Signature-256"), body) {
		http.Error(w, "Assinatura inválida", 401)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err = a.GitHub.Webhook(ctx, r.Header.Get("X-GitHub-Delivery"), r.Header.Get("X-GitHub-Event"), body); err != nil {
		slog.Error("webhook failed", "error", err)
		http.Error(w, "Falha ao processar webhook; reenvie este evento", 500)
		return
	}
	w.WriteHeader(204)
}
func issueURL(key string, number int32) string { return "/issues/" + issue.Reference(key, number) }
func releaseURL(id int64) string               { return fmt.Sprintf("/releases/%d", id) }
