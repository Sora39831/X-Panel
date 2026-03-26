# Cloudflare Turnstile Registration Captcha Design

> **Goal:** Replace the arithmetic captcha on the registration page with Cloudflare Turnstile explicit verification, so that registration can only proceed after Turnstile verification passes.

**Architecture:** Turnstile script loads in `<body>` with `render=explicit`. On tab switch to register, a polling loop waits for the `turnstile` global to be ready, then calls `turnstile.render()`. The returned token is sent to the backend, which verifies it server-side via the Turnstile siteverify API.

**Tech Stack:** Go (Gin), Vue.js, Cloudflare Turnstile (explicit render mode)

---

## Files to Modify

| File | Changes |
|------|---------|
| `web/html/login.html` | Add Turnstile script, replace captcha UI with widget div, update Vue data/methods |
| `web/controller/index.go` | Replace `RegisterForm` fields, add `verifyTurnstile()`, update register handler |

No new files created. No i18n changes needed (reuse existing `wrongCaptcha` key).

---

## Frontend Changes (`web/html/login.html`)

### 1. Add Turnstile Script

After `{{ template "page/body_start" .}}` (inside `<body>`), insert:

```html
<script src="https://challenges.cloudflare.com/turnstile/v0.js?render=explicit" async defer></script>
```

**Why explicit mode:** The register tab is hidden by default (`activeTab: "login"`). Auto-render would target a hidden element. Explicit mode lets us render on demand.

**Why after body_start:** Script tags between `</head>` and `<body>` are invalid HTML and get silently dropped by browsers (root cause of previous failure).

### 2. Replace Captcha UI

Remove the arithmetic captcha input (`regCaptcha` field + placeholder showing `captchaQuestion`). Replace with:

```html
<a-form-item>
  <div id="turnstile-widget"></div>
</a-form-item>
```

### 3. Vue Data Changes

Remove:
- `captcha` (user input for arithmetic answer)
- `captchaAnswer` (pre-computed correct answer)
- `captchaQuestion` (displayed "a + b = ?" string)

Add:
- `turnstileToken: ""` — token received from Turnstile widget callback
- `turnstileRendered: false` — guard to prevent double-render

### 4. Vue Methods Changes

**onTabChange(key):** Replace `activeTab = $event` binding with method call. On switch to register tab, call `this.$nextTick(() => this.renderTurnstile())`.

**renderTurnstile():** Polling function that checks `typeof turnstile !== 'undefined'`. If undefined, retries after 200ms. Once ready and not yet rendered, calls `turnstile.render('#turnstile-widget', { sitekey: '0x4AAAAAACwR0LBVK-2kqbSa', callback: (token) => { this.turnstileToken = token; } })` and sets `turnstileRendered = true`.

**register():**
- Remove old captcha match check (`captcha !== captchaAnswer`)
- Add check: `if (!this.turnstileToken)` → show `wrongCaptcha` error
- POST body: `{ email, password, turnstileToken }` (replacing `captcha`/`captchaAnswer`)
- On success: call `turnstile.reset()` and set `this.turnstileToken = ""`

**mounted():** Remove `generateCaptcha()` call.

### 5. Template Binding Change

`@change="activeTab = $event"` → `@change="onTabChange"`

---

## Backend Changes (`web/controller/index.go`)

### 1. RegisterForm Struct

```go
type RegisterForm struct {
    Email          string `json:"email" form:"email"`
    Password       string `json:"password" form:"password"`
    TurnstileToken string `json:"turnstileToken" form:"turnstileToken"`
}
```

### 2. Turnstile Verification Function

```go
const turnstileSecretKey = "0x4AAAAAACwR0BwMTZCdnEg_0NWHEBa6RwE"

func verifyTurnstile(token string, remoteIP string) bool {
    resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
        "secret":   {turnstileSecretKey},
        "response": {token},
        "remoteip": {remoteIP},
    })
    // decode JSON, return result.Success
}
```

Required imports to add: `net/url`, `encoding/json`.

### 3. Register Handler

Replace captcha validation block with:

```go
if form.TurnstileToken == "" || !verifyTurnstile(form.TurnstileToken, clientIP) {
    pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.register.toasts.wrongCaptcha"))
    return
}
```

---

## Data Flow

```
User clicks Register tab
  → onTabChange('register')
    → $nextTick → renderTurnstile()
      → polls: typeof turnstile !== 'undefined'?
        → No: retry 200ms later
        → Yes: turnstile.render('#turnstile-widget', { sitekey, callback })
          → User completes Turnstile challenge
            → callback(token) → this.turnstileToken = token

User clicks Submit
  → register()
    → token empty? → show error
    → POST { email, password, turnstileToken } → /register
      → verifyTurnstile(token, clientIP)
        → POST https://challenges.cloudflare.com/turnstile/v0/siteverify
          → { success: true/false }
        → true: proceed to userService.Register()
        → false: show wrongCaptcha error
```

---

## Verification

1. `go build ./...` passes
2. Deploy to VPS, open login page → Turnstile script loads (check Network tab)
3. Switch to Register tab → Turnstile widget renders
4. Complete Turnstile → submit registration → success
5. Open Register tab → do NOT complete Turnstile → submit → shows captcha error
6. Login as registered user → userinfo page shows correctly
