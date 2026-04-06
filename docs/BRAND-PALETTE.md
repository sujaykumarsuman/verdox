# Verdox Design Token Reference

Complete design system tokens for the Verdox platform.
Aesthetic: minimal and soothing, warm neutrals, teal accent.

---

## 1. Color Tokens

### Core Palette

| Token | Hex | Usage |
|---|---|---|
| `--accent` | `#1C6D74` | Primary buttons, active states, links |
| `--accent-light` | `#248F98` | Hover states, secondary highlights |
| `--accent-dark` | `#155459` | Pressed states |
| `--accent-subtle` | `#E8F4F5` | Tinted backgrounds, selected rows, soft highlights |

### Backgrounds

| Token | Hex | Mode | Usage |
|---|---|---|---|
| `--bg-primary` | `#FAFAF8` | Light | Page background |
| `--bg-secondary` | `#F0EDE6` | Light | Card/panel backgrounds |
| `--bg-tertiary` | `#E8E5DD` | Light | Nested elements, table stripes, inset panels |
| `--bg-primary-dark` | `#1A1A1A` | Dark | Page background |
| `--bg-secondary-dark` | `#242424` | Dark | Card/panel backgrounds |
| `--bg-tertiary-dark` | `#2E2E2E` | Dark | Nested elements, table stripes, inset panels |

### Text

| Token | Hex | Mode | Usage |
|---|---|---|---|
| `--text-primary` | `#1A1A1A` | Light | Main body text |
| `--text-secondary` | `#6B6B6B` | Light | Muted text, labels, placeholders |
| `--text-primary-dark` | `#E8E8E8` | Dark | Main body text |
| `--text-secondary-dark` | `#9A9A9A` | Dark | Muted text, labels, placeholders |

### Semantic

| Token | Hex | Usage |
|---|---|---|
| `--success` | `#2D8A4E` | Pass, success states, positive indicators |
| `--danger` | `#C93B3B` | Fail, error states, destructive actions |
| `--warning` | `#D4910A` | Warnings, caution indicators |

### Borders

| Token | Hex | Mode | Usage |
|---|---|---|---|
| `--border` | `#E2DFD8` | Light | Borders, dividers, separators |
| `--border-dark` | `#333333` | Dark | Borders, dividers, separators |

### Utility

| Token | Value | Usage |
|---|---|---|
| `--focus-ring` | `0 0 0 2px #FAFAF8, 0 0 0 4px #1C6D74` | Accessibility focus outline (double-ring) |
| `--overlay` | `rgba(0, 0, 0, 0.45)` | Modal/dialog backdrop |
| `--disabled` | `#B8B8B8` | Disabled button text, disabled input text |
| `--disabled-bg` | `#EDEDEB` | Disabled button/input background (light) |
| `--disabled-bg-dark` | `#2A2A2A` | Disabled button/input background (dark) |

### CSS Custom Properties (Light Mode Defaults)

```css
:root {
  /* Accent */
  --accent: #1C6D74;
  --accent-light: #248F98;
  --accent-dark: #155459;
  --accent-subtle: #E8F4F5;

  /* Backgrounds */
  --bg-primary: #FAFAF8;
  --bg-secondary: #F0EDE6;
  --bg-tertiary: #E8E5DD;

  /* Text */
  --text-primary: #1A1A1A;
  --text-secondary: #6B6B6B;

  /* Semantic */
  --success: #2D8A4E;
  --danger: #C93B3B;
  --warning: #D4910A;

  /* Borders */
  --border: #E2DFD8;

  /* Utility */
  --focus-ring: 0 0 0 2px #FAFAF8, 0 0 0 4px #1C6D74;
  --overlay: rgba(0, 0, 0, 0.45);
  --disabled: #B8B8B8;
  --disabled-bg: #EDEDEB;
}
```

### CSS Custom Properties (Dark Mode)

```css
[data-theme="dark"] {
  --bg-primary: #1A1A1A;
  --bg-secondary: #242424;
  --bg-tertiary: #2E2E2E;

  --text-primary: #E8E8E8;
  --text-secondary: #9A9A9A;

  --border: #333333;

  --focus-ring: 0 0 0 2px #1A1A1A, 0 0 0 4px #248F98;
  --overlay: rgba(0, 0, 0, 0.65);
  --disabled: #5A5A5A;
  --disabled-bg: #2A2A2A;
}
```

---

## 2. Typography Scale

### Font Families

| Token | Value | Usage |
|---|---|---|
| `--font-display` | `"DM Serif Display", Georgia, serif` | Display headings |
| `--font-body` | `"DM Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif` | Body text, UI elements |
| `--font-mono` | `"JetBrains Mono", "Fira Code", "Consolas", monospace` | Code blocks, log output, terminal |

### Google Fonts Import

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link
  href="https://fonts.googleapis.com/css2?family=DM+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400;1,500&family=DM+Serif+Display&family=JetBrains+Mono:wght@400;500&display=swap"
  rel="stylesheet"
/>
```

### Type Scale

| Token | Font Family | Size / Line Height | Weight | Letter Spacing | Usage |
|---|---|---|---|---|---|
| `--type-display` | DM Serif Display | 36px / 44px | 400 | -0.02em | Page titles, hero headings |
| `--type-h1` | DM Serif Display | 30px / 38px | 400 | -0.01em | Section headings |
| `--type-h2` | DM Serif Display | 24px / 32px | 400 | -0.01em | Subsection headings |
| `--type-h3` | DM Sans | 20px / 28px | 600 | -0.01em | Card titles, group labels |
| `--type-h4` | DM Sans | 18px / 26px | 600 | 0 | Sub-labels, sidebar headers |
| `--type-body-lg` | DM Sans | 18px / 28px | 400 | 0 | Lead paragraphs, feature text |
| `--type-body` | DM Sans | 16px / 24px | 400 | 0 | Default body text |
| `--type-body-sm` | DM Sans | 14px / 20px | 400 | 0 | Secondary text, table cells |
| `--type-caption` | DM Sans | 12px / 16px | 500 | 0.02em | Timestamps, footnotes, badges |
| `--type-code` | JetBrains Mono | 14px / 20px | 400 | 0 | Inline code, log output |

### CSS

```css
:root {
  --font-display: "DM Serif Display", Georgia, serif;
  --font-body: "DM Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-mono: "JetBrains Mono", "Fira Code", "Consolas", monospace;
}

.type-display { font-family: var(--font-display); font-size: 36px; line-height: 44px; font-weight: 400; letter-spacing: -0.02em; }
.type-h1     { font-family: var(--font-display); font-size: 30px; line-height: 38px; font-weight: 400; letter-spacing: -0.01em; }
.type-h2     { font-family: var(--font-display); font-size: 24px; line-height: 32px; font-weight: 400; letter-spacing: -0.01em; }
.type-h3     { font-family: var(--font-body);    font-size: 20px; line-height: 28px; font-weight: 600; letter-spacing: -0.01em; }
.type-h4     { font-family: var(--font-body);    font-size: 18px; line-height: 26px; font-weight: 600; letter-spacing: 0; }
.type-body-lg { font-family: var(--font-body);   font-size: 18px; line-height: 28px; font-weight: 400; letter-spacing: 0; }
.type-body   { font-family: var(--font-body);    font-size: 16px; line-height: 24px; font-weight: 400; letter-spacing: 0; }
.type-body-sm { font-family: var(--font-body);   font-size: 14px; line-height: 20px; font-weight: 400; letter-spacing: 0; }
.type-caption { font-family: var(--font-body);   font-size: 12px; line-height: 16px; font-weight: 500; letter-spacing: 0.02em; }
.type-code   { font-family: var(--font-mono);    font-size: 14px; line-height: 20px; font-weight: 400; letter-spacing: 0; }
```

---

## 3. Spacing Scale

Base unit: **4px**.

| Token | Value | Common Use |
|---|---|---|
| `--space-1` | `4px` | Inline icon gap, tight padding |
| `--space-2` | `8px` | Badge padding, compact gaps |
| `--space-3` | `12px` | Input padding, small card padding |
| `--space-4` | `16px` | Default element gap, button padding-x |
| `--space-5` | `20px` | Card padding (compact) |
| `--space-6` | `24px` | Card padding (default), section gap |
| `--space-8` | `32px` | Section spacing, sidebar padding |
| `--space-10` | `40px` | Page margin (mobile) |
| `--space-12` | `48px` | Large section dividers |
| `--space-16` | `64px` | Page margin (desktop), hero spacing |

### CSS

```css
:root {
  --space-1:  4px;
  --space-2:  8px;
  --space-3:  12px;
  --space-4:  16px;
  --space-5:  20px;
  --space-6:  24px;
  --space-8:  32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;
}
```

---

## 4. Shadow Tokens

All shadows use black at low opacity to maintain the warm, soothing feel.

| Token | Value | Usage |
|---|---|---|
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.04)` | Subtle lift: inputs, small elements |
| `--shadow-md` | `0 2px 4px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)` | Hover cards, dropdowns |
| `--shadow-lg` | `0 4px 12px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04)` | Modals, floating panels |
| `--shadow-card` | `0 1px 3px rgba(0,0,0,0.04)` | Default card resting state |

### CSS

```css
:root {
  --shadow-sm:   0 1px 2px rgba(0,0,0,0.04);
  --shadow-md:   0 2px 4px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04);
  --shadow-lg:   0 4px 12px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04);
  --shadow-card: 0 1px 3px rgba(0,0,0,0.04);
}
```

---

## 5. Border Radius Tokens

Rounded but not bubbly.

| Token | Value | Usage |
|---|---|---|
| `--radius-sm` | `4px` | Inputs, badges, code blocks |
| `--radius-md` | `6px` | Buttons, dropdowns |
| `--radius-lg` | `8px` | Cards, panels, sidebars |
| `--radius-xl` | `12px` | Modals, large dropdowns, popovers |
| `--radius-full` | `9999px` | Avatars, pill badges, toggles |

### CSS

```css
:root {
  --radius-sm:   4px;
  --radius-md:   6px;
  --radius-lg:   8px;
  --radius-xl:   12px;
  --radius-full: 9999px;
}
```

---

## 6. Transition Tokens

Smooth, unobtrusive motion.

| Token | Value | Usage |
|---|---|---|
| `--duration-fast` | `150ms` | Checkbox toggles, icon swaps |
| `--duration-normal` | `200ms` | Button hovers, color changes |
| `--duration-slow` | `300ms` | Page transitions, sidebar collapse, modals |
| `--easing-default` | `cubic-bezier(0.4, 0, 0.2, 1)` | General-purpose easing |
| `--easing-in` | `cubic-bezier(0.4, 0, 1, 1)` | Elements exiting view |
| `--easing-out` | `cubic-bezier(0, 0, 0.2, 1)` | Elements entering view |

### Common Patterns

```css
/* Button hover */
.btn {
  transition: background-color var(--duration-normal) var(--easing-default),
              box-shadow var(--duration-normal) var(--easing-default);
}

/* Page/route transition */
.page-enter {
  transition: opacity var(--duration-slow) var(--easing-out),
              transform var(--duration-slow) var(--easing-out);
}

/* Sidebar collapse */
.sidebar {
  transition: width var(--duration-slow) var(--easing-default);
}
```

### CSS

```css
:root {
  --duration-fast:   150ms;
  --duration-normal: 200ms;
  --duration-slow:   300ms;
  --easing-default:  cubic-bezier(0.4, 0, 0.2, 1);
  --easing-in:       cubic-bezier(0.4, 0, 1, 1);
  --easing-out:      cubic-bezier(0, 0, 0.2, 1);
}
```

---

## 7. Breakpoints

Mobile-first. Apply styles at the breakpoint and above.

| Token | Value | Target |
|---|---|---|
| `--bp-sm` | `640px` | Large phones (landscape) |
| `--bp-md` | `768px` | Tablets |
| `--bp-lg` | `1024px` | Small laptops, landscape tablets |
| `--bp-xl` | `1280px` | Desktops |
| `--bp-2xl` | `1536px` | Large/wide monitors |

### Usage

```css
/* Tablet and up */
@media (min-width: 768px) { ... }

/* Desktop and up */
@media (min-width: 1024px) { ... }
```

---

## 8. Dark Mode Mapping

Complete mapping from light mode tokens to dark mode values.

| Property | Light Token | Light Value | Dark Value |
|---|---|---|---|
| Page background | `--bg-primary` | `#FAFAF8` | `#1A1A1A` |
| Card background | `--bg-secondary` | `#F0EDE6` | `#242424` |
| Nested/inset background | `--bg-tertiary` | `#E8E5DD` | `#2E2E2E` |
| Main text | `--text-primary` | `#1A1A1A` | `#E8E8E8` |
| Muted text | `--text-secondary` | `#6B6B6B` | `#9A9A9A` |
| Borders | `--border` | `#E2DFD8` | `#333333` |
| Disabled background | `--disabled-bg` | `#EDEDEB` | `#2A2A2A` |
| Disabled text | `--disabled` | `#B8B8B8` | `#5A5A5A` |
| Focus ring | `--focus-ring` | `0 0 0 2px #FAFAF8, 0 0 0 4px #1C6D74` | `0 0 0 2px #1A1A1A, 0 0 0 4px #248F98` |
| Overlay | `--overlay` | `rgba(0,0,0,0.45)` | `rgba(0,0,0,0.65)` |
| Accent | `--accent` | `#1C6D74` | `#1C6D74` (unchanged) |
| Accent light | `--accent-light` | `#248F98` | `#248F98` (unchanged) |
| Accent dark | `--accent-dark` | `#155459` | `#155459` (unchanged) |
| Accent subtle | `--accent-subtle` | `#E8F4F5` | `#1A2E30` |
| Success | `--success` | `#2D8A4E` | `#2D8A4E` (unchanged) |
| Danger | `--danger` | `#C93B3B` | `#C93B3B` (unchanged) |
| Warning | `--warning` | `#D4910A` | `#D4910A` (unchanged) |
| Card shadow | `--shadow-card` | `0 1px 3px rgba(0,0,0,0.04)` | `0 1px 3px rgba(0,0,0,0.2)` |

### Implementation

Toggle dark mode by setting `data-theme="dark"` on the `<html>` element via a sun/moon toggle button.

```typescript
function toggleTheme() {
  const html = document.documentElement;
  const current = html.getAttribute("data-theme");
  const next = current === "dark" ? "light" : "dark";
  html.setAttribute("data-theme", next);
  localStorage.setItem("verdox-theme", next);
}

// On load: restore saved preference or respect system setting
const saved = localStorage.getItem("verdox-theme");
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
document.documentElement.setAttribute(
  "data-theme",
  saved ?? (prefersDark ? "dark" : "light")
);
```

---

## 9. Tailwind CSS Configuration

Map all design tokens into `tailwind.config.ts`.

```typescript
import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  darkMode: ["class", '[data-theme="dark"]'],
  theme: {
    screens: {
      sm: "640px",
      md: "768px",
      lg: "1024px",
      xl: "1280px",
      "2xl": "1536px",
    },
    extend: {
      colors: {
        accent: {
          DEFAULT: "#1C6D74",
          light: "#248F98",
          dark: "#155459",
          subtle: "#E8F4F5",
        },
        bg: {
          primary: "var(--bg-primary)",
          secondary: "var(--bg-secondary)",
          tertiary: "var(--bg-tertiary)",
        },
        text: {
          primary: "var(--text-primary)",
          secondary: "var(--text-secondary)",
        },
        success: "#2D8A4E",
        danger: "#C93B3B",
        warning: "#D4910A",
        border: "var(--border)",
        disabled: {
          DEFAULT: "var(--disabled)",
          bg: "var(--disabled-bg)",
        },
      },
      fontFamily: {
        display: ['"DM Serif Display"', "Georgia", "serif"],
        body: ['"DM Sans"', "-apple-system", "BlinkMacSystemFont", '"Segoe UI"', "sans-serif"],
        mono: ['"JetBrains Mono"', '"Fira Code"', '"Consolas"', "monospace"],
      },
      fontSize: {
        display: ["36px", { lineHeight: "44px", letterSpacing: "-0.02em", fontWeight: "400" }],
        h1: ["30px", { lineHeight: "38px", letterSpacing: "-0.01em", fontWeight: "400" }],
        h2: ["24px", { lineHeight: "32px", letterSpacing: "-0.01em", fontWeight: "400" }],
        h3: ["20px", { lineHeight: "28px", letterSpacing: "-0.01em", fontWeight: "600" }],
        h4: ["18px", { lineHeight: "26px", letterSpacing: "0em", fontWeight: "600" }],
        "body-lg": ["18px", { lineHeight: "28px", letterSpacing: "0em", fontWeight: "400" }],
        body: ["16px", { lineHeight: "24px", letterSpacing: "0em", fontWeight: "400" }],
        "body-sm": ["14px", { lineHeight: "20px", letterSpacing: "0em", fontWeight: "400" }],
        caption: ["12px", { lineHeight: "16px", letterSpacing: "0.02em", fontWeight: "500" }],
        code: ["14px", { lineHeight: "20px", letterSpacing: "0em", fontWeight: "400" }],
      },
      spacing: {
        1: "4px",
        2: "8px",
        3: "12px",
        4: "16px",
        5: "20px",
        6: "24px",
        8: "32px",
        10: "40px",
        12: "48px",
        16: "64px",
      },
      borderRadius: {
        sm: "4px",
        md: "6px",
        lg: "8px",
        xl: "12px",
        full: "9999px",
      },
      boxShadow: {
        sm: "0 1px 2px rgba(0,0,0,0.04)",
        md: "0 2px 4px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)",
        lg: "0 4px 12px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04)",
        card: "0 1px 3px rgba(0,0,0,0.04)",
      },
      transitionDuration: {
        fast: "150ms",
        normal: "200ms",
        slow: "300ms",
      },
      transitionTimingFunction: {
        default: "cubic-bezier(0.4, 0, 0.2, 1)",
        in: "cubic-bezier(0.4, 0, 1, 1)",
        out: "cubic-bezier(0, 0, 0.2, 1)",
      },
    },
  },
  plugins: [],
};

export default config;
```

---

## 10. Component Token Usage Guide

### Buttons

| Variant | Background | Text | Border | Hover BG | Active BG | Radius | Shadow |
|---|---|---|---|---|---|---|---|
| Primary | `--accent` | `#FFFFFF` | none | `--accent-light` | `--accent-dark` | `--radius-md` | none |
| Secondary | transparent | `--accent` | `1px solid --accent` | `--accent-subtle` | `--accent-subtle` | `--radius-md` | none |
| Ghost | transparent | `--text-primary` | none | `--bg-secondary` | `--bg-tertiary` | `--radius-md` | none |
| Danger | `--danger` | `#FFFFFF` | none | `#B33232` | `#9E2B2B` | `--radius-md` | none |
| Disabled (any) | `--disabled-bg` | `--disabled` | none | -- | -- | `--radius-md` | none |

All buttons: `padding: var(--space-2) var(--space-4)`, `font: var(--type-body-sm)`, `font-weight: 500`, `transition: var(--duration-normal) var(--easing-default)`.

### Cards

| Property | Token |
|---|---|
| Background | `--bg-secondary` |
| Border | `1px solid var(--border)` |
| Shadow | `--shadow-card` |
| Hover shadow | `--shadow-md` |
| Radius | `--radius-lg` |
| Padding | `--space-6` |

### Inputs

| Property | Token |
|---|---|
| Background | `--bg-primary` |
| Border | `1px solid var(--border)` |
| Border (focus) | `1px solid var(--accent)` |
| Focus ring | `--focus-ring` |
| Text | `--text-primary` |
| Placeholder | `--text-secondary` |
| Radius | `--radius-sm` |
| Padding | `var(--space-3) var(--space-3)` |
| Font | `--type-body` |
| Disabled BG | `--disabled-bg` |
| Disabled text | `--disabled` |

### Sidebar

| Property | Token |
|---|---|
| Background | `--bg-secondary` |
| Border right | `1px solid var(--border)` |
| Width (expanded) | `260px` |
| Width (collapsed) | `64px` |
| Padding | `--space-4` |
| Nav item padding | `var(--space-2) var(--space-3)` |
| Nav item radius | `--radius-md` |
| Active item BG | `--accent-subtle` |
| Active item text | `--accent` |
| Inactive item text | `--text-secondary` |
| Hover item BG | `--bg-tertiary` |
| Collapse transition | `--duration-slow` |

### TopBar

| Property | Token |
|---|---|
| Background | `--bg-primary` |
| Border bottom | `1px solid var(--border)` |
| Height | `56px` |
| Padding | `0 var(--space-6)` |
| Title font | `--type-h4` |
| Shadow | none (border-only separation) |

### Badges

| Variant | Background | Text | Border |
|---|---|---|---|
| Pass / Success | `#E6F4EC` | `--success` | none |
| Fail / Error | `#FDEAEA` | `--danger` | none |
| Pending / Warning | `#FEF3D9` | `--warning` | none |
| Neutral | `--bg-tertiary` | `--text-secondary` | none |

All badges: `padding: var(--space-1) var(--space-2)`, `font: var(--type-caption)`, `font-weight: 500`, `radius: var(--radius-sm)`.

Dark mode badge backgrounds adjust to lower-opacity tints:

| Variant (Dark) | Background |
|---|---|
| Pass / Success | `rgba(45, 138, 78, 0.15)` |
| Fail / Error | `rgba(201, 59, 59, 0.15)` |
| Pending / Warning | `rgba(212, 145, 10, 0.15)` |
| Neutral | `--bg-tertiary-dark` |

### Modals

| Property | Token |
|---|---|
| Overlay | `--overlay` |
| Background | `--bg-primary` |
| Border | `1px solid var(--border)` |
| Shadow | `--shadow-lg` |
| Radius | `--radius-xl` |
| Padding | `--space-8` |
| Max width | `560px` |
| Enter transition | `--duration-slow` + `--easing-out` |
| Exit transition | `--duration-normal` + `--easing-in` |

---

## Quick Reference: All CSS Custom Properties

```css
:root {
  /* Colors - Accent */
  --accent: #1C6D74;
  --accent-light: #248F98;
  --accent-dark: #155459;
  --accent-subtle: #E8F4F5;

  /* Colors - Backgrounds */
  --bg-primary: #FAFAF8;
  --bg-secondary: #F0EDE6;
  --bg-tertiary: #E8E5DD;

  /* Colors - Text */
  --text-primary: #1A1A1A;
  --text-secondary: #6B6B6B;

  /* Colors - Semantic */
  --success: #2D8A4E;
  --danger: #C93B3B;
  --warning: #D4910A;

  /* Colors - Borders */
  --border: #E2DFD8;

  /* Colors - Utility */
  --focus-ring: 0 0 0 2px #FAFAF8, 0 0 0 4px #1C6D74;
  --overlay: rgba(0, 0, 0, 0.45);
  --disabled: #B8B8B8;
  --disabled-bg: #EDEDEB;

  /* Typography */
  --font-display: "DM Serif Display", Georgia, serif;
  --font-body: "DM Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-mono: "JetBrains Mono", "Fira Code", "Consolas", monospace;

  /* Spacing */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.04);
  --shadow-md: 0 2px 4px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04);
  --shadow-lg: 0 4px 12px rgba(0,0,0,0.08), 0 2px 4px rgba(0,0,0,0.04);
  --shadow-card: 0 1px 3px rgba(0,0,0,0.04);

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-xl: 12px;
  --radius-full: 9999px;

  /* Transitions */
  --duration-fast: 150ms;
  --duration-normal: 200ms;
  --duration-slow: 300ms;
  --easing-default: cubic-bezier(0.4, 0, 0.2, 1);
  --easing-in: cubic-bezier(0.4, 0, 1, 1);
  --easing-out: cubic-bezier(0, 0, 0.2, 1);
}

[data-theme="dark"] {
  --bg-primary: #1A1A1A;
  --bg-secondary: #242424;
  --bg-tertiary: #2E2E2E;
  --text-primary: #E8E8E8;
  --text-secondary: #9A9A9A;
  --border: #333333;
  --accent-subtle: #1A2E30;
  --focus-ring: 0 0 0 2px #1A1A1A, 0 0 0 4px #248F98;
  --overlay: rgba(0, 0, 0, 0.65);
  --disabled: #5A5A5A;
  --disabled-bg: #2A2A2A;
  --shadow-card: 0 1px 3px rgba(0,0,0,0.2);
}
```
