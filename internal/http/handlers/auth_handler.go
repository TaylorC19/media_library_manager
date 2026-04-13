package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"media_library_manager/internal/config"
	"media_library_manager/internal/http/middleware"
	authsvc "media_library_manager/internal/service/auth"
	"media_library_manager/internal/views"
)

type AuthHandler struct {
	cfg     config.Config
	render  *views.Renderer
	service *authsvc.Service
}

type authJSONResponse struct {
	OK       bool     `json:"ok"`
	Redirect string   `json:"redirect,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func NewAuthHandler(cfg config.Config, render *views.Renderer, service *authsvc.Service) *AuthHandler {
	return &AuthHandler{cfg: cfg, render: render, service: service}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if middleware.CurrentUser(r.Context()) != nil {
		locale := middleware.LocaleFromContext(r.Context())
		http.Redirect(w, r, "/"+locale+"/", http.StatusFound)
		return
	}

	data := h.baseAuthData(r, "Login", nil, nil)
	data["ContentTemplate"] = "pages/login.content"
	h.consumeFlash(w, r, data)
	h.renderTemplate(w, "pages/login.html", data)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if middleware.CurrentUser(r.Context()) != nil {
		locale := middleware.LocaleFromContext(r.Context())
		http.Redirect(w, r, "/"+locale+"/", http.StatusFound)
		return
	}

	data := h.baseAuthData(r, "Register", nil, nil)
	data["ContentTemplate"] = "pages/register.content"
	h.consumeFlash(w, r, data)
	h.renderTemplate(w, "pages/register.html", data)
}

func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	_, token, err := h.service.Login(r.Context(), username, password, r.UserAgent(), clientIP(r))
	if err != nil {
		msgKey := "auth.errors.invalidCredentials"
		if !errors.Is(err, authsvc.ErrInvalidCredentials) {
			msgKey = "auth.errors.serverError"
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusBadRequest, authJSONResponse{
				OK:     false,
				Errors: []string{authMessage(locale, msgKey)},
			})
			return
		}
		data := h.baseAuthData(r, "Login", map[string]string{"username": username}, []string{msgKey})
		data["ContentTemplate"] = "pages/login.content"
		h.renderTemplate(w, "pages/login.html", data)
		return
	}

	h.setSessionCookie(w, token)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, authJSONResponse{
			OK:       true,
			Redirect: "/" + locale + "/",
		})
		return
	}
	http.Redirect(w, r, "/"+locale+"/", http.StatusFound)
}

func (h *AuthHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	_, token, err := h.service.Register(r.Context(), username, password)
	if err != nil {
		msgKey := "auth.errors.serverError"
		if errors.Is(err, authsvc.ErrInvalidInput) {
			msgKey = "auth.errors.invalidRegisterInput"
		} else if errors.Is(err, authsvc.ErrUsernameTaken) {
			msgKey = "auth.errors.usernameTaken"
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusBadRequest, authJSONResponse{
				OK:     false,
				Errors: []string{authMessage(locale, msgKey)},
			})
			return
		}
		data := h.baseAuthData(r, "Register", map[string]string{"username": username}, []string{msgKey})
		data["ContentTemplate"] = "pages/register.content"
		h.renderTemplate(w, "pages/register.html", data)
		return
	}

	h.setSessionCookie(w, token)
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    middleware.EncodeFlash("common.flash.info", "auth.flash.registered"),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10,
	})
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, authJSONResponse{
			OK:       true,
			Redirect: "/" + locale + "/",
		})
		return
	}
	http.Redirect(w, r, "/"+locale+"/", http.StatusFound)
}

func (h *AuthHandler) LogoutSubmit(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())

	if cookie, err := r.Cookie(h.cfg.SessionCookieName); err == nil {
		_ = h.service.LogoutByToken(r.Context(), cookie.Value)
	}

	h.clearSessionCookie(w)
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    middleware.EncodeFlash("common.flash.info", "auth.flash.loggedOut"),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10,
	})

	http.Redirect(w, r, "/"+locale+"/login", http.StatusFound)
}

func (h *AuthHandler) baseAuthData(r *http.Request, title string, values map[string]string, formErrors []string) map[string]any {
	locale := middleware.LocaleFromContext(r.Context())
	if values == nil {
		values = map[string]string{}
	}
	if formErrors == nil {
		formErrors = []string{}
	}

	return map[string]any{
		"Title":      title + " - Media Library Manager",
		"PageTitle":  title,
		"Locale":     locale,
		"Values":     values,
		"FormErrors": formErrors,
	}
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  timeNowUTC().Add(h.cfg.SessionTTL()),
		MaxAge:   int(h.cfg.SessionTTL().Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) consumeFlash(w http.ResponseWriter, r *http.Request, data map[string]any) {
	c, err := r.Cookie("flash")
	if err != nil {
		return
	}
	flash := middleware.DecodeFlash(c.Value)
	if flash != nil {
		data["Flash"] = flash
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) renderTemplate(w http.ResponseWriter, page string, data map[string]any) {
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return strings.TrimSpace(host)
}

var timeNowUTC = func() time.Time {
	return time.Now().UTC()
}

func wantsJSON(r *http.Request) bool {
	xrw := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Requested-With")))
	if xrw == "fetch" || xrw == "xmlhttprequest" {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/json")
}

func writeJSON(w http.ResponseWriter, status int, payload authJSONResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func authMessage(locale, key string) string {
	isJA := strings.EqualFold(locale, "ja")
	switch key {
	case "auth.errors.invalidCredentials":
		if isJA {
			return "ユーザー名またはパスワードが正しくありません。"
		}
		return "Invalid username or password."
	case "auth.errors.invalidRegisterInput":
		if isJA {
			return "パスワードは8文字以上、ユーザー名は空にしないでください。"
		}
		return "Use at least 8 characters for the password and a non-empty username."
	case "auth.errors.usernameTaken":
		if isJA {
			return "このユーザー名は既に登録されています。"
		}
		return "That username is already registered."
	default:
		if isJA {
			return "エラーが発生しました。しばらくしてから再度お試しください。"
		}
		return "Something went wrong. Please try again."
	}
}
