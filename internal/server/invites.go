package server

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ramon/trackline/internal/auth"
	"github.com/ramon/trackline/internal/database"
	"github.com/ramon/trackline/web/pages"
)

func (a *App) createInvite(w http.ResponseWriter, r *http.Request) error {
	if User(r).Role != "admin" {
		return auth.ErrForbidden
	}
	token := auth.Token()
	_, err := a.Pool.Exec(r.Context(), "INSERT INTO workspace_invites(token_hash,workspace_id,created_by,expires_at) VALUES($1,$2,$3,$4)", auth.TokenHash(token), User(r).WorkspaceID, User(r).ID, time.Now().Add(7*24*time.Hour))
	if err != nil {
		return err
	}
	redirect(w, r, "/settings?invite="+token)
	return nil
}

func (a *App) inviteInfo(r *http.Request) (pages.InviteData, int64, error) {
	token := r.PathValue("token")
	if len(token) != 43 {
		return pages.InviteData{}, 0, pgx.ErrNoRows
	}
	data := pages.InviteData{CSRF: csrf(r), Token: token, AppName: "1984"}
	var workspaceID int64
	err := a.Pool.QueryRow(r.Context(), `SELECT w.id,w.name,w.app_name,w.logo IS NOT NULL FROM workspace_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.token_hash=$1 AND i.expires_at>now()`, auth.TokenHash(token)).Scan(&workspaceID, &data.WorkspaceName, &data.AppName, &data.HasLogo)
	if err != nil {
		return data, 0, err
	}
	if cookie, cookieErr := r.Cookie(a.cookieName("session")); cookieErr == nil {
		var name string
		if queryErr := a.Pool.QueryRow(r.Context(), `SELECT u.name FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, auth.TokenHash(cookie.Value)).Scan(&name); queryErr == nil {
			data.LoggedIn = true
			data.CurrentName = name
		}
	}
	return data, workspaceID, nil
}

func (a *App) invitePage(w http.ResponseWriter, r *http.Request) error {
	data, _, err := a.inviteInfo(r)
	if err != nil {
		return err
	}
	return render(w, r, pages.Invite(data))
}

func (a *App) acceptInvite(w http.ResponseWriter, r *http.Request) error {
	_, workspaceID, err := a.inviteInfo(r)
	if err != nil {
		return err
	}
	var name, email, hash string
	if r.FormValue("accept_current") == "1" {
		cookie, cookieErr := r.Cookie(a.cookieName("session"))
		if cookieErr != nil {
			return auth.ErrCredentials
		}
		err = a.Pool.QueryRow(r.Context(), `SELECT u.name,u.email,u.password_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, auth.TokenHash(cookie.Value)).Scan(&name, &email, &hash)
		if err != nil {
			return auth.ErrCredentials
		}
	} else {
		email = strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		address, mailErr := mail.ParseAddress(email)
		if mailErr != nil || address.Address != email || len(email) > 254 {
			return errors.New("digite um email válido")
		}
		err = a.Pool.QueryRow(r.Context(), "SELECT name,password_hash FROM users WHERE lower(email)=lower($1) AND active LIMIT 1", email).Scan(&name, &hash)
		if err == nil {
			if !auth.CheckPassword(hash, r.FormValue("password")) {
				return auth.ErrCredentials
			}
		} else if errors.Is(err, pgx.ErrNoRows) {
			name = strings.TrimSpace(r.FormValue("name"))
			if name == "" || len([]rune(name)) > 120 {
				return errors.New("informe seu nome")
			}
			hash, err = auth.HashPassword(r.FormValue("password"))
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	var userID int64
	err = a.Pool.QueryRow(r.Context(), "SELECT id FROM users WHERE workspace_id=$1 AND lower(email)=lower($2)", workspaceID, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = a.Pool.QueryRow(r.Context(), "INSERT INTO users(workspace_id,name,email,password_hash,role) VALUES($1,$2,$3,$4,'member') RETURNING id", workspaceID, name, email, hash).Scan(&userID)
	} else if err == nil {
		_, err = a.Pool.Exec(r.Context(), "UPDATE users SET active=true,role='member',updated_at=now() WHERE id=$1", userID)
	}
	if err != nil {
		return err
	}
	token := auth.Token()
	if err = a.Queries.CreateSession(r.Context(), database.CreateSessionParams{TokenHash: auth.TokenHash(token), UserID: userID, ExpiresAt: database.Timestamp(time.Now().Add(30 * 24 * time.Hour))}); err != nil {
		return err
	}
	a.cookie(w, "session", token, 30*86400)
	a.cookie(w, "csrf", auth.Token(), 30*86400)
	redirect(w, r, "/")
	return nil
}

func (a *App) switchWorkspace(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return errors.New("workspace inválido")
	}
	var userID int64
	err = a.Pool.QueryRow(r.Context(), "SELECT id FROM users WHERE workspace_id=$1 AND lower(email)=lower($2) AND active", workspaceID, User(r).Email).Scan(&userID)
	if err != nil {
		return auth.ErrForbidden
	}
	cookie, err := r.Cookie(a.cookieName("session"))
	if err != nil {
		return err
	}
	_, err = a.Pool.Exec(r.Context(), "UPDATE sessions SET user_id=$1 WHERE token_hash=$2", userID, auth.TokenHash(cookie.Value))
	if err != nil {
		return err
	}
	redirect(w, r, "/")
	return nil
}

func (a *App) updateUserRole(w http.ResponseWriter, r *http.Request) error {
	if User(r).Role != "admin" {
		return auth.ErrForbidden
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	tag, err := a.Pool.Exec(r.Context(), "UPDATE users SET role='admin',updated_at=now() WHERE id=$1 AND workspace_id=$2", id, User(r).WorkspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	redirect(w, r, "/settings")
	return nil
}

func (a *App) resetUserPassword(w http.ResponseWriter, r *http.Request) error {
	if User(r).Role != "admin" {
		return auth.ErrForbidden
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(r.FormValue("password"))
	if err != nil {
		return err
	}
	tag, err := a.Pool.Exec(r.Context(), "UPDATE users SET password_hash=$1,updated_at=now() WHERE id=$2 AND workspace_id=$3 AND active", hash, id, User(r).WorkspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	_, err = a.Pool.Exec(r.Context(), "DELETE FROM sessions WHERE user_id=$1", id)
	if err != nil {
		return err
	}
	redirect(w, r, "/settings")
	return nil
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) error {
	if User(r).Role != "admin" {
		return auth.ErrForbidden
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		return err
	}
	if id == User(r).ID {
		return errors.New("não é possível remover seu próprio usuário")
	}
	tag, err := a.Pool.Exec(r.Context(), "UPDATE users SET active=false,role='member',updated_at=now() WHERE id=$1 AND workspace_id=$2 AND active", id, User(r).WorkspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	_, err = a.Pool.Exec(r.Context(), "DELETE FROM sessions WHERE user_id=$1", id)
	if err != nil {
		return err
	}
	redirect(w, r, "/settings")
	return nil
}
