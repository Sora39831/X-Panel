package controller

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"x-ui/logger"
	"x-ui/web/service"
	"x-ui/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9_.@]+$`)

type LoginForm struct {
	Username    	string `json:"username" form:"username"`
	Password    	string `json:"password" form:"password"`
	TwoFactorCode	string `json:"twoFactorCode" form:"twoFactorCode"`
}

type RegisterForm struct {
	Email          string `json:"email" form:"email"`
	Password       string `json:"password" form:"password"`
	TurnstileToken string `json:"turnstileToken" form:"turnstileToken"`
}

const turnstileSecretKey = "0x4AAAAAACwR0BwMTZCdnEg_0NWHEBa6RwE"

func verifyTurnstile(token string, remoteIP string) bool {
	if token == "" {
		return false
	}
	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {turnstileSecretKey},
		"response": {token},
		"remoteip": {remoteIP},
	})
	if err != nil {
		logger.Warningf("Turnstile verification request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Warningf("Failed to decode Turnstile response: %v", err)
		return false
	}
	return result.Success
}

type IndexController struct {
	BaseController

	settingService service.SettingService
	userService    service.UserService
	tgbot          service.Tgbot
}

// IP 限流器：同一 IP 每分钟最多 5 次注册请求
var registerRateLimiter = struct {
	sync.Mutex
	records map[string][]time.Time
}{records: make(map[string][]time.Time)}

func checkRegisterRateLimit(ip string) bool {
	registerRateLimiter.Lock()
	defer registerRateLimiter.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// 清理过期记录
	var valid []time.Time
	for _, t := range registerRateLimiter.records[ip] {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= 5 {
		return false
	}
	registerRateLimiter.records[ip] = append(valid, now)

	// 定期清理长时间未使用的 IP 记录
	if len(registerRateLimiter.records) > 100 {
		for k, v := range registerRateLimiter.records {
			if len(v) == 0 || v[len(v)-1].Before(windowStart) {
				delete(registerRateLimiter.records, k)
			}
		}
	}

	return true
}

func NewIndexController(g *gin.RouterGroup) *IndexController {
	a := &IndexController{}
	a.initRouter(g)
	return a
}

func (a *IndexController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.index)
	g.POST("/login", a.login)
	g.GET("/logout", a.logout)
	g.POST("/getTwoFactorEnable", a.getTwoFactorEnable)
	g.POST("/register", a.register)
}

func (a *IndexController) index(c *gin.Context) {
	if session.IsLogin(c) {
		c.Redirect(http.StatusTemporaryRedirect, "panel/")
		return
	}
	html(c, "login.html", "pages.login.title", nil)
}

func (a *IndexController) login(c *gin.Context) {
	var form LoginForm

	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.invalidFormData"))
		return
	}
	if form.Username == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyUsername"))
		return
	}
	if form.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyPassword"))
		return
	}

	user := a.userService.CheckUser(form.Username, form.Password, form.TwoFactorCode)
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	safeUser := template.HTMLEscapeString(form.Username)
	safePass := template.HTMLEscapeString(form.Password)

	if user == nil {
		logger.Warningf("wrong username: \"%s\", password: \"%s\", IP: \"%s\"", safeUser, safePass, getRemoteIp(c))
		a.tgbot.UserLoginNotify(safeUser, safePass, getRemoteIp(c), timeStr, 0)
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.wrongUsernameOrPassword"))
		return
	}

	logger.Infof("%s logged in successfully, Ip Address: %s\n", safeUser, getRemoteIp(c))
	a.tgbot.UserLoginNotify(safeUser, ``, getRemoteIp(c), timeStr, 1)

	sessionMaxAge, err := a.settingService.GetSessionMaxAge()
	if err != nil {
		logger.Warning("Unable to get session's max age from DB")
	}

	session.SetMaxAge(c, sessionMaxAge*60)
	session.SetLoginUser(c, user)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session: ", err)
		return
	}

	logger.Infof("%s logged in successfully", safeUser)
	jsonMsg(c, I18nWeb(c, "pages.login.toasts.successLogin"), nil)
}

func (a *IndexController) logout(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user != nil {
		logger.Infof("%s logged out successfully", user.Username)
	}
	session.ClearSession(c)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session after clearing:", err)
	}
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
}

func (a *IndexController) getTwoFactorEnable(c *gin.Context) {
	status, err := a.settingService.GetTwoFactorEnable()
	if err == nil {
		jsonObj(c, status, nil)
	}
}

func (a *IndexController) register(c *gin.Context) {
	var form RegisterForm

	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.invalidFormData"))
		return
	}

	// IP 限流检查
	clientIP := getRemoteIp(c)
	if !checkRegisterRateLimit(clientIP) {
		pureJsonMsg(c, http.StatusTooManyRequests, false, I18nWeb(c, "pages.register.toasts.tooManyRequests"))
		return
	}

	// 验证 email
	if form.Email == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.emptyEmail"))
		return
	}
	if len(form.Email) < 4 || len(form.Email) > 64 {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.emailLengthError"))
		return
	}
	if !emailRegex.MatchString(form.Email) || strings.Count(form.Email, "@") > 1 {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.invalidEmail"))
		return
	}

	// 验证密码
	if form.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.emptyPassword"))
		return
	}
	if len(form.Password) < 6 {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.passwordTooShort"))
		return
	}
	hasLetter := false
	hasDigit := false
	for _, ch := range form.Password {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasLetter = true
		}
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.passwordTooWeak"))
		return
	}

	// Cloudflare Turnstile 验证
	if !verifyTurnstile(form.TurnstileToken, clientIP) {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.wrongCaptcha"))
		return
	}

	// 调用注册服务
	err := a.userService.Register(form.Email, form.Password)
	if err != nil {
		logger.Warningf("Registration failed for email %s: %v", form.Email, err)
		if strings.Contains(err.Error(), "already exists") {
			pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.emailAlreadyExists"))
		} else {
			pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.registerFailed"))
		}
		return
	}

	logger.Infof("New user registered: %s", form.Email)
	jsonMsg(c, I18nWeb(c, "pages.register.toasts.successRegister"), nil)
}
