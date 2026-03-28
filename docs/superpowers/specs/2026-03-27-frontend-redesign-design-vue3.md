# X-Panel Frontend Redesign — Apple HIG + Vue 3

## Context

X-Panel's admin UI currently uses Vue.js 2 + Ant Design Vue 1.x with server-side Go templates, a teal color scheme, and 3-state theme (light/dark/ultra-dark). The goal is a complete frontend rewrite adopting Apple HIG design language, Vue 3, Naive UI, and modern build tooling while preserving the Go backend with minimal changes.

## Design Decisions

- **Architecture:** Vite + Vue 3 + TypeScript SPA; build output embedded via Go `embed.FS`
- **Component library:** Naive UI (Vue 3 native, Apple HIG-adjacent design)
- **Design language:** Apple Human Interface Guidelines
- **Font:** HarmonyOS Sans (self-hosted woff2, CJK+Latin)
- **HDR:** Progressive enhancement — display-p3 wide gamut with sRGB fallback
  - Detection: `@media (color-gamut: rec2020)` for Windows 11 HDR, `@media (color-gamut: p3)` for other wide-gamut devices
  - Color values: `color(display-p3 ...)` (subset of rec2020, safe and visible improvement)
- **Target browsers:** Windows 11 Edge 120+, Android Edge
- **Theme:** 2-state (light/dark) with system preference auto-detect, localStorage persistence
- **Internationalization:** Chinese (zh-CN) + English (en-US) only
- **Navigation:** Desktop = hover-expand overlay sidebar (no layout shift); Mobile = bottom tab bar
- **Motion:** Smooth micro-animations via CSS transitions; respect `prefers-reduced-motion`

## Go Backend Changes (Minimal)

| File | Change |
|------|--------|
| `web/web.go` | Add `assets/dist/*` to embed paths |
| `web/html/common/page.html` | Replace 10+ `<script>` tags with single `dist/index.js` + `dist/index.css` |
| `web/controller/` | No changes — all API endpoints remain identical |

## Project Structure

```
web/
  frontend/                          # Vite project root
    src/
      main.ts                        # createApp entry
      App.vue                        # Root component (router outlet)
      router/index.ts                # Vue Router
      stores/                        # Pinia stores
        theme.ts                     # Dark/light toggle
        hdr.ts                       # HDR state
        auth.ts                      # Login state
        i18n.ts                      # Language toggle
      composables/
        useHDR.ts                    # HDR detection
        useTheme.ts                  # Theme management
        useResponsive.ts             # Responsive breakpoints
        useApi.ts                    # API wrapper
      components/
        layout/
          AppSidebar.vue             # Desktop hover sidebar
          AppBottomNav.vue           # Mobile bottom tab bar
          AppHeader.vue              # Top bar
        common/
          StatusBadge.vue
          TrafficBar.vue
          ConfirmDialog.vue
        forms/
          InboundForm.vue
          ClientForm.vue
          ProtocolSettings.vue
        modals/
          QRModal.vue
          LinkModal.vue
          ImportExportModal.vue
      views/
        LoginView.vue
        DashboardView.vue
        InboundsView.vue
        SettingsView.vue
        XrayView.vue
        ServersView.vue
        UserInfoView.vue
      assets/
        styles/
          variables.css              # CSS custom properties
          typography.css             # Font definitions
          themes.css                 # Light/dark + HDR
          animations.css             # Transitions
          naive-overrides.css        # Naive UI customizations
      i18n/
        zh-CN.ts
        en-US.ts
      types/
        index.ts
      api/
        index.ts                     # Existing API endpoints wrapper
    vite.config.ts
    tsconfig.json
    package.json
  assets/
    dist/                            # Vite build output
    fonts/                           # HarmonyOS-Sans.woff2 (existing)
```

## Design System

### Color Tokens — Light Mode (sRGB)

```css
:root {
  --color-primary: #007AFF;
  --color-primary-hover: #0066D6;
  --color-primary-active: #0055B3;
  --color-primary-bg: rgba(0, 122, 255, 0.08);

  --color-success: #34C759;
  --color-warning: #FF9500;
  --color-error: #FF3B30;

  --color-bg: #F2F2F7;
  --color-surface: #FFFFFF;
  --color-surface-secondary: #F9F9FB;
  --color-border: rgba(60, 60, 67, 0.12);

  --color-text-primary: #1C1C1E;
  --color-text-secondary: #8E8E93;
  --color-text-tertiary: #C7C7CC;
}
```

### Color Tokens — Dark Mode (sRGB)

```css
html.dark {
  --color-primary: #0A84FF;
  --color-primary-hover: #409CFF;
  --color-primary-active: #0066D6;
  --color-primary-bg: rgba(10, 132, 255, 0.12);

  --color-bg: #000000;
  --color-surface: #1C1C1E;
  --color-surface-secondary: #2C2C2E;
  --color-border: rgba(84, 84, 88, 0.36);

  --color-text-primary: rgba(255, 255, 255, 0.92);
  --color-text-secondary: rgba(255, 255, 255, 0.56);
  --color-text-tertiary: rgba(255, 255, 255, 0.28);
}
```

### Color Tokens — HDR (display-p3)

```css
/* rec2020 detection covers Windows 11 HDR + future devices */
/* p3 detection covers macOS, high-end Android OLED */
/* Both use display-p3 color values (safe, visible improvement) */

@media (color-gamut: rec2020) {
  :root {
    --color-primary: color(display-p3 0 0.478 1);
    --color-primary-hover: color(display-p3 0 0.4 0.84);
    --color-success: color(display-p3 0.204 0.78 0.349);
    --color-warning: color(display-p3 1 0.584 0);
    --color-error: color(display-p3 1 0.231 0.188);
    --color-bg: color(display-p3 0.949 0.949 0.969);
  }
  html.dark {
    --color-primary: color(display-p3 0.039 0.518 1);
    --color-primary-hover: color(display-p3 0.251 0.612 1);
    --color-bg: color(display-p3 0 0 0);
    --color-surface: color(display-p3 0.11 0.11 0.118);
    --color-text-primary: color(display-p3 1 1 1 / 0.92);
  }
}

@media (color-gamut: p3) {
  :root {
    --color-primary: color(display-p3 0 0.478 1);
    --color-primary-hover: color(display-p3 0 0.4 0.84);
    --color-success: color(display-p3 0.204 0.78 0.349);
    --color-warning: color(display-p3 1 0.584 0);
    --color-error: color(display-p3 1 0.231 0.188);
    --color-bg: color(display-p3 0.949 0.949 0.969);
  }
  html.dark {
    --color-primary: color(display-p3 0.039 0.518 1);
    --color-primary-hover: color(display-p3 0.251 0.612 1);
    --color-bg: color(display-p3 0 0 0);
    --color-surface: color(display-p3 0.11 0.11 0.118);
    --color-text-primary: color(display-p3 1 1 1 / 0.92);
  }
}
```

### Dark Mode Readability (WCAG AA Compliance)

| Element | Value | Contrast (vs #000) |
|---------|-------|-------------------|
| Primary text | `rgba(255,255,255,0.92)` | ~16.6:1 |
| Secondary text | `rgba(255,255,255,0.56)` | ~9.3:1 |
| Tertiary text | `rgba(255,255,255,0.28)` | ~4.6:1 |
| Primary color | `#0A84FF` | ~5.2:1 |

All interactive text elements meet WCAG AA (contrast ratio >= 4.5:1).

### Typography

```css
@font-face {
  font-family: 'HarmonyOS Sans';
  src: url('/assets/fonts/HarmonyOS-Sans-Regular.woff2') format('woff2');
  font-weight: 400;
  font-display: swap;
}
@font-face {
  font-family: 'HarmonyOS Sans';
  src: url('/assets/fonts/HarmonyOS-Sans-Medium.woff2') format('woff2');
  font-weight: 500;
  font-display: swap;
}

:root {
  --font-family: 'HarmonyOS Sans', -apple-system, BlinkMacSystemFont,
                 'Segoe UI', Roboto, sans-serif;
  --font-size-xs: 11px;
  --font-size-sm: 13px;
  --font-size-base: 15px;
  --font-size-lg: 17px;
  --font-size-xl: 20px;
  --font-size-2xl: 24px;
  --font-size-3xl: 32px;
  --font-weight-regular: 400;
  --font-weight-medium: 500;
  --font-weight-semibold: 600;
  --line-height-tight: 1.2;
  --line-height-normal: 1.5;
}
```

## Layout & Navigation

### Desktop (>=768px): Hover-Expand Sidebar

- Default: 64px icon column
- Hover: expands to 240px overlay on top of content (no layout shift)
- Background: `backdrop-filter: blur(20px)` + semi-transparent
- Auto-collapse after 300ms mouse leave (with debounce)
- Items: Dashboard, Inbounds, Settings, Xray Config, Servers, Navigation, UserInfo

### Mobile (<768px): Bottom Tab Bar

- 5 tabs: Dashboard, Inbounds, Settings, Xray, More
- "More" expands to include Servers, Navigation, UserInfo
- Height: 49px + `safe-area-inset-bottom` (notch safe)
- Background: frosted glass

### Responsive Breakpoints

```
< 768px   -> Mobile (bottom tab bar, full-width layout)
768-1024  -> Tablet (sidebar + content, content max-width)
> 1024px  -> Desktop (sidebar + content, fluid)
```

## Animation System

```css
:root {
  --duration-fast: 150ms;
  --duration-normal: 250ms;
  --duration-slow: 400ms;
  --ease-out: cubic-bezier(0.25, 0.46, 0.45, 0.94);
  --ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1);
}

@media (prefers-reduced-motion: reduce) {
  :root {
    --duration-fast: 0ms;
    --duration-normal: 0ms;
    --duration-slow: 0ms;
  }
}
```

Key animations:
- Page transitions: fade (opacity)
- Card hover: translateY(-1px) + shadow
- Button active: scale(0.97)
- Sidebar: opacity + blur transition
- Modal: scale(0.95 -> 1) + opacity

## Naive UI Overrides

```typescript
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#007AFF',
    borderRadius: '10px',
    borderRadiusSmall: '6px',
    fontFamily: 'var(--font-family)',
    fontSize: '15px',
  },
  Button: { borderRadiusMedium: '10px', heightMedium: '36px' },
  Card: { borderRadius: '16px' },
  Modal: { borderRadius: '16px' },
  Input: { borderRadius: '10px' },
  DataTable: { borderRadius: '12px', thColor: 'transparent' },
}
```

## HDR Detection Composable

```typescript
// composables/useHDR.ts
export function useHDR() {
  const colorGamut = ref<'srgb' | 'p3'>('srgb')

  const update = () => {
    // Windows 11 HDR reports rec2020; high-end OLED reports p3
    // Both indicate wide gamut support
    if (matchMedia('(color-gamut: rec2020)').matches
        || matchMedia('(color-gamut: p3)').matches) {
      colorGamut.value = 'p3'
    } else {
      colorGamut.value = 'srgb'
    }
  }

  onMounted(() => {
    update()
    matchMedia('(color-gamut: rec2020)').addEventListener('change', update)
    matchMedia('(color-gamut: p3)').addEventListener('change', update)
  })

  return { colorGamut, isWideGamut: computed(() => colorGamut.value === 'p3') }
}
```

## Pages Overview

| Route | View | Key Features |
|-------|------|-------------|
| `/` | LoginView | Login form + optional registration, Turnstile |
| `/panel/` | DashboardView | System stats, traffic overview, status cards |
| `/panel/inbounds` | InboundsView | Inbound list, CRUD, client management |
| `/panel/settings` | SettingsView | Panel config (12 sub-sections) |
| `/panel/xray` | XrayView | JSON editor (CodeMirror) |
| `/panel/servers` | ServersView | Remote VPS management |
| `/panel/navigation` | NavigationView | External links/resources |
| `/panel/userinfo` | UserInfoView | User traffic info (non-admin) |

## Build & Deploy

```bash
# Development
cd web/frontend && npm run dev    # Vite dev server with HMR

# Production
cd web/frontend && npm run build  # Output to web/assets/dist/
go build                          # embed.FS includes dist/
```

## Non-Goals (Out of Scope)

- TypeScript migration of existing Go backend
- Adding new API endpoints
- Changing authentication mechanism
- Server-side rendering (SSR)
- PWA/offline support
