# X-Panel Frontend Redesign — Apple HIG Design Language

## Context

X-Panel's admin UI currently uses a teal color scheme (`#008771`), 3-state theme (light/dark/ultra-dark), system font stack, and basic Ant Design Vue styling. The goal is to transform it into an Apple HIG-compliant interface while keeping Vue.js 2 + Ant Design Vue 1.x as the framework.

Key drivers:
- More polished, professional appearance matching modern admin panels
- HDR-capable displays becoming mainstream — progressive color enhancement
- Responsive design for phone/tablet/desktop usage
- Single clean dark mode (Apple HIG has one dark, not ultra-dark)

## Design Decisions

- **Approach A (Full CSS Variable System)** — rewrite `custom.min.css` as the foundation
- **Primary color:** Apple system blue (`#007AFF` light / `#0A84FF` dark) replacing teal
- **Font:** HarmonyOS Sans (self-hosted woff2) with system fallback stack
- **HDR:** Progressive enhancement via `@media (dynamic-range: high)` with `color(display-p3 ...)` values
- **Theme:** 2-state (light/dark) with system preference auto-detect & dynamic listening, localStorage persistence
- **Breakpoints:** Phone <768px, Tablet 768–1024px, Desktop >1024px
- **Sidebar:** Desktop = frosted glass collapsible sider (with solid fallback); Tablet = icon-only rail; Phone = bottom tab bar

---

## Files to Modify

| File | Responsibility |
|------|---------------|
| `web/assets/css/custom.min.css` | Complete rewrite — CSS variables, Apple HIG colors, responsive breakpoints, HDR media query, component overrides, custom scrollbars |
| `web/assets/fonts/` (new dir) | HarmonyOS Sans Regular + Medium woff2 files |
| `web/html/common/page.html` | Add `@font-face` declarations, inject blocking dark-mode detection script in `<head>` to prevent FOUC |
| `web/html/component/aThemeSwitch.html` | Simplify to 2-state toggle, remove ultra-dark, add dynamic system-preference listener |
| `web/html/component/aSidebar.html` | Restyle with frosted glass material, add tablet rail mode, mobile bottom tab bar (with memory leak prevention) |
| `web/html/login.html` | Restyle card/wave/button with Apple blue, keep zoom animation |
| `web/html/index.html` | Card spacing, typography tweaks (minimal) |
| `web/html/inbounds.html` | Table spacing on tablet (minimal) |
| `web/html/settings.html` | Form spacing (minimal) |
| `web/html/xray.html` | CodeMirror theme alignment (minimal) |
| `web/html/servers.html` | Card grid responsive (minimal) |
| `web/html/navigation.html` | Card layout responsive (minimal) |
| `web/html/userinfo.html` | Profile card responsive (minimal) |

---

## Section 1: CSS Foundation (`custom.min.css`)

### 1.1 CSS Custom Properties — Light Mode

```css
:root {
  /* Apple System Blue */
  --color-primary: #007AFF;
  --color-primary-hover: #0066D6;
  --color-primary-active: #0055B3;
  --color-primary-light: rgba(0, 122, 255, 0.1);
  --color-primary-lighter: rgba(0, 122, 255, 0.05);

  /* Apple HIG Backgrounds */
  --color-background: #F2F2F7;
  --color-surface: #FFFFFF;
  --color-surface-secondary: #F2F2F7;
  --color-surface-tertiary: #E5E5EA;

  /* Apple HIG Text */
  --color-text-primary: #000000;
  --color-text-secondary: rgba(0, 0, 0, 0.6);
  --color-text-tertiary: rgba(0, 0, 0, 0.3);
  --color-text-quaternary: rgba(0, 0, 0, 0.18);

  /* Separators */
  --color-separator: rgba(60, 60, 67, 0.12);
  --color-separator-opaque: #C6C6C8;

  /* Fills */
  --color-fill: rgba(120, 120, 128, 0.2);
  --color-fill-secondary: rgba(120, 120, 128, 0.16);
  --color-fill-tertiary: rgba(120, 120, 128, 0.12);
  --color-fill-quaternary: rgba(120, 120, 128, 0.08);

  /* Semantic Colors */
  --color-success: #34C759;
  --color-warning: #FF9500;
  --color-danger: #FF3B30;
  --color-info: #5856D6;

  /* Layout */
  --sidebar-width: 200px;
  --sidebar-collapsed-width: 64px;
  --border-radius-sm: 8px;
  --border-radius-md: 12px;
  --border-radius-lg: 16px;
  --border-radius-xl: 20px;

  /* Elevation (Apple-style shadows) */
  --shadow-1: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06);
  --shadow-2: 0 4px 6px rgba(0, 0, 0, 0.04), 0 2px 4px rgba(0, 0, 0, 0.06);
  --shadow-3: 0 10px 20px rgba(0, 0, 0, 0.06), 0 4px 8px rgba(0, 0, 0, 0.04);

  /* Transitions */
  --transition-fast: 150ms cubic-bezier(0.25, 0.1, 0.25, 1);
  --transition-base: 200ms cubic-bezier(0.25, 0.1, 0.25, 1);
  --transition-slow: 300ms cubic-bezier(0.25, 0.1, 0.25, 1);

  /* Sidebar material */
  --sidebar-bg-solid: rgba(255, 255, 255, 0.95); /* Fallback for older browsers */
  --sidebar-bg: rgba(255, 255, 255, 0.72);
  --sidebar-border: rgba(0, 0, 0, 0.06);
}
```

### 1.2 Dark Mode Variables

```css
html.dark {
  --color-background: #000000;
  --color-surface: #1C1C1E;
  --color-surface-secondary: #2C2C2E;
  --color-surface-tertiary: #3A3A3C;

  --color-text-primary: #FFFFFF;
  --color-text-secondary: rgba(255, 255, 255, 0.6);
  --color-text-tertiary: rgba(255, 255, 255, 0.3);
  --color-text-quaternary: rgba(255, 255, 255, 0.18);

  --color-separator: rgba(84, 84, 88, 0.65);
  --color-separator-opaque: #38383A;

  --color-fill: rgba(120, 120, 128, 0.36);
  --color-fill-secondary: rgba(120, 120, 128, 0.32);
  --color-fill-tertiary: rgba(120, 120, 128, 0.24);
  --color-fill-quaternary: rgba(120, 120, 128, 0.18);

  --color-primary: #0A84FF;

  --shadow-1: 0 1px 3px rgba(0, 0, 0, 0.2);
  --shadow-2: 0 4px 8px rgba(0, 0, 0, 0.3);
  --shadow-3: 0 10px 20px rgba(0, 0, 0, 0.4);

  --sidebar-bg-solid: rgba(28, 28, 30, 0.95);
  --sidebar-bg: rgba(28, 28, 30, 0.72);
  --sidebar-border: rgba(255, 255, 255, 0.06);
}
```

### 1.3 HDR Progressive Enhancement

```css
@media (dynamic-range: high) {
  :root {
    --color-primary: color(display-p3 0 0.478 1);
    --color-success: color(display-p3 0.204 0.78 0.349);
    --color-warning: color(display-p3 1 0.584 0);
    --color-danger: color(display-p3 1 0.231 0.188);
    --color-info: color(display-p3 0.345 0.337 0.843);
  }
  html.dark {
    --color-primary: color(display-p3 0.039 0.518 1);
  }
}
```

### 1.4 Base Resets & Safe Area

```css
*, *::before, *::after { box-sizing: border-box; }

html, body {
  height: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  color: var(--color-text-primary);
  font-family: 'HarmonyOS Sans', -apple-system, BlinkMacSystemFont, 'Vazirmatn', 'Segoe UI', Roboto, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  background-color: var(--color-background);
  /* Notch/Dynamic Island support for landscape */
  padding-left: env(safe-area-inset-left);
  padding-right: env(safe-area-inset-right);
}

/* Custom Apple-style Scrollbars */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: var(--color-separator-opaque);
  border-radius: 10px;
}
::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-tertiary);
}
* {
  scrollbar-width: thin;
  scrollbar-color: var(--color-separator-opaque) transparent;
}
```

### 1.5 Ant Design Overrides (Apple-style)

*Note: AntD 1.x uses Less. To avoid override hell, map CSS variables meticulously. Use `!important` judiciously where AntD applies inline styles or deep nested specificity.*

Key overrides needed:
- **Cards:** `border-radius: var(--border-radius-lg)`, `background: var(--color-surface)`, `border: 1px solid var(--color-separator)`
- **Tables:** Remove heavy borders, use `--color-separator` for light dividers, rounded rows
- **Buttons:** Apple pill-style primary buttons, subtle fills for default buttons
- **Inputs:** `border-radius: 10px`, fill-based backgrounds (`--color-fill-quaternary`)
- **Modals:** Frosted glass backdrop, `border-radius: var(--border-radius-xl)`
- **Tags:** Subtle fills matching semantic colors
- **Menu items:** Subtle hover fills, `border-radius: var(--border-radius-sm)`, no animated gradient selection (replace with solid `--color-primary-light` fill)
- **Switches:** Apple-style (iOS) with `--color-primary` when checked

### 1.6 Responsive Breakpoints

```css
/* Phone: < 768px */
@media (max-width: 767px) {
  .ant-layout-sider { display: none !important; }
  .ant-card, .ant-alert-error { margin: 8px; }
  .ant-tabs { margin: 8px; padding: 8px; }
  .ant-modal-body { padding: 16px; }
}

/* Tablet: 768px – 1024px */
@media (min-width: 768px) and (max-width: 1024px) {
  .ant-layout-sider { width: 64px !important; max-width: 64px !important; flex: 0 0 64px !important; }
  .ant-layout-sider-collapsed { width: 64px !important; max-width: 64px !important; flex: 0 0 64px !important; }
  /* Tables: allow horizontal scroll, reduce padding */
  .ant-table-thead > tr > th, .ant-table-tbody > tr > td { padding: 8px 6px; }
}
```

### 1.7 Phone Bottom Tab Bar

New CSS component for phone navigation (includes iOS safe area inset):

```css
.mobile-tab-bar {
  display: none;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: calc(49px + env(safe-area-inset-bottom));
  padding-bottom: env(safe-area-inset-bottom);
  background: var(--sidebar-bg-solid); /* Fallback */
  border-top: 0.5px solid var(--color-separator);
  z-index: 100;
  justify-content: space-around;
  align-items: center;
}

@supports (backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px)) {
  .mobile-tab-bar {
    background: var(--sidebar-bg);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
  }
}

@media (max-width: 767px) {
  .mobile-tab-bar { display: flex; }
  #app { padding-bottom: calc(49px + env(safe-area-inset-bottom)); }
}
```

---

## Section 2: Font Setup

### 2.1 Font Files

Place in `web/assets/fonts/`:
- `HarmonyOS-Sans-Regular.woff2` (weight 400)
- `HarmonyOS-Sans-Medium.woff2` (weight 500)

### 2.2 @font-face in `page.html`

Add to the `<style>` block in `page/head_start`:

```css
@font-face {
  font-display: swap;
  font-family: 'HarmonyOS Sans';
  font-weight: 400;
  font-style: normal;
  src: url('{{ .base_path }}assets/fonts/HarmonyOS-Sans-Regular.woff2') format('woff2');
}
@font-face {
  font-display: swap;
  font-family: 'HarmonyOS Sans';
  font-weight: 500;
  font-style: normal;
  src: url('{{ .base_path }}assets/fonts/HarmonyOS-Sans-Medium.woff2') format('woff2');
}
```

---

## Section 3: Sidebar (`aSidebar.html`)

### 3.1 Desktop (>1024px)
- **Glassmorphism with Graceful Degradation:**
  ```css
  .ant-layout-sider {
    background: var(--sidebar-bg-solid) !important; /* Fallback */
    border-right: 1px solid var(--sidebar-border);
  }
  @supports (backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px)) {
    .ant-layout-sider {
      background: var(--sidebar-bg) !important;
      backdrop-filter: saturate(180%) blur(20px);
      -webkit-backdrop-filter: saturate(180%) blur(20px);
    }
  }
  ```
- Keep collapsible behavior (200px ↔ 64px)

### 3.2 Tablet (768–1024px)
- Force collapsed (64px rail) — icons only, no text labels
- `overflow: hidden` to prevent text peeking

### 3.3 Bottom Tab Bar HTML & Vue logic

Add to sidebar template — rendered conditionally via Vue:

```html
<div class="mobile-tab-bar" v-if="isMobile">
  <div v-for="tab in primaryTabs" :key="tab.key"
       class="mobile-tab-item"
       :class="{ active: activeTab[0] === tab.key }"
       @click="openLink(tab.key)">
    <a-icon :type="tab.icon" />
    <span class="mobile-tab-label">[[ tab.shortTitle ]]</span>
  </div>
  <div class="mobile-tab-item" @click="toggleDrawer()">
    <a-icon type="ellipsis" />
    <span class="mobile-tab-label">{{ i18n "menu.more" }}</span>
  </div>
</div>
```

Vue Data & Lifecycle (with Memory Leak Prevention):
```javascript
data() {
  return {
    primaryTabs: [
      { key: '...', icon: 'dashboard', shortTitle: '{{ i18n "menu.dashboard"}}' },
      { key: '...', icon: 'user', shortTitle: '{{ i18n "menu.inbounds"}}' },
      { key: '...', icon: 'setting', shortTitle: '{{ i18n "menu.settings"}}' },
      { key: '...', icon: 'tool', shortTitle: '{{ i18n "menu.xray"}}' },
    ],
    isMobile: window.innerWidth < 768,
  };
},
mounted() {
  this.checkMobileResize = () => { this.isMobile = window.innerWidth < 768; };
  window.addEventListener('resize', this.checkMobileResize, { passive: true });
},
beforeDestroy() {
  window.removeEventListener('resize', this.checkMobileResize);
}
```

---

## Section 4: Theme Switcher (`aThemeSwitch.html`)

### 4.1 FOUC Prevention & Dynamic Detection (Inject in `<head>`)

To prevent the dreaded Flash of Unstyled Content (FOUC) when loading the dark theme, add this blocking script into the `<head>` of `page.html` (before CSS/Vue loads):

```html
<script>
  (function() {
    const stored = localStorage.getItem('dark-mode');
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    // Check localStorage first, fallback to system preference
    const isDark = stored === 'true' || (stored === null && mql.matches);
    
    if (isDark) {
      document.documentElement.classList.add('dark');
    }

    // Dynamic system theme listener (if user hasn't forced a preference)
    mql.addEventListener('change', (e) => {
      if (localStorage.getItem('dark-mode') === null) {
        if (e.matches) {
          document.documentElement.classList.add('dark');
        } else {
          document.documentElement.classList.remove('dark');
        }
      }
    });
  })();
</script>
```

### 4.2 Simplification in Vue Component

Remove:
- Ultra-dark toggle and related code (`isUltra`, `toggleUltra()`, `animationsOffUltra()`)

Keep:
- `toggleTheme()` method: simply toggles `document.documentElement.classList.toggle('dark')` and updates `localStorage.setItem('dark-mode', boolean)`.
- Replace submenu with a single iOS-style switch in sidebar.

---

## Section 5: Login Page (`login.html`)

### 5.1 Background & Card
- Background: `var(--color-background)`
- Card: `var(--color-surface)` with `border-radius: var(--border-radius-xl)` (20px)
- `box-shadow: var(--shadow-2)`
- Dark mode border: `border: 1px solid var(--color-separator)`
- Keep animated wave SVG & zoom text animation. Change wave fill colors to match Apple palette (Light: `#DBEAFE`, Dark: `var(--color-surface)`).

### 5.2 Buttons
- Replace teal gradient sweep with Apple blue solid (`var(--color-primary)`)
- `border-radius: 14px` (Apple pill style)
- Hover: `background: var(--color-primary-hover)`
- Remove wave-btn-bg gradient animation in dark mode.

---

## Section 6: Other Pages

Changes are minimal — primarily CSS-driven via the foundation. 

- **Dashboard (`index.html`):** Update hardcoded teal references.
- **Inbounds (`inbounds.html`):** Handled by CSS overrides.
- **Settings (`settings.html`):** Handled by CSS overrides.
- **Xray Config (`xray.html`):** CodeMirror theme: update gutter/selection colors via CSS.
- **Servers/Navigation/UserInfo:** Grid and Card layout handled by CSS overrides.

---

## Section 7: Translation Keys

No new translation keys needed for the redesign — existing i18n keys are reused. Add one new key:

### English (`translate.en_US.toml`)
```toml
"menu.more" = "More"
```

### Chinese (`translate.zh_CN.toml`)
```toml
"menu.more" = "更多"
```

Remove `menu.ultraDark` key (no longer used).

---

## Micro-interactions & Performance

To avoid severe layout thrashing and painting performance drops, **never** apply `* { transition: all }`. Instead, use a temporary class during the toggle:

```javascript
// Inside toggleTheme()
document.body.classList.add('theme-transitioning');
setTimeout(() => document.body.classList.remove('theme-transitioning'), 300);
```

```css
/* In custom.min.css */
body.theme-transitioning, 
body.theme-transitioning * {
  transition: background-color 300ms ease, border-color 300ms ease, color 300ms ease !important;
}
```

Other interactions:
- **Menu/Card hover:** `transition: background-color var(--transition-fast)`
- **Button press:** `transform: scale(0.97)` on `:active`
- **Bottom tab bar:** `transition: color var(--transition-fast)`
- Easing for all: `cubic-bezier(0.25, 0.1, 0.25, 1)`

---

## Verification

1. `go build ./...` — must pass
2. Visual check in Chrome:
   - **Desktop (>1024px):** Sidebar expands/collapses, frosted glass visible.
   - **Tablet (800px):** Sidebar shows icon rail, tables allow horizontal scroll.
   - **Phone (375px):** Bottom tab bar visible (respects safe-area on iOS), sidebar drawer works.
3. Theme toggle: light → dark → light, **ensure no FOUC (flash of white screen) on initial page load in dark mode.**
4. System preference: `localStorage.clear()`, change OS theme to Dark/Light, panel should follow immediately.
5. Resize listener check: Resize window rapidly, check Vue devtools to ensure no performance lag (memory leak prevented).
6. HDR check: colors should be visibly richer on XDR/P3 displays.