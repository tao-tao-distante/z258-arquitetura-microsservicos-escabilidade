# Atelier UI

## Direction

- Artifact: responsive task-management application.
- Positioning: utilitarian, editorial, quiet, focused.
- Aesthetic essence: editorial workbench clarity.
- Source of truth: Figma page `V2 · ATELIÊ / Frontend editorial`, node `7:2`.
- Signature move: dark pine navigation and progress surfaces against warm paper.

## Foundations

- Display: Archivo, bold, compact hierarchy.
- Body: DM Sans, regular and medium.
- Palette: paper `#f5f2ea`, raised paper `#fffcf5`, pine `#183d35`, soft pine `#264d42`, lime `#ddf08a`, ink `#26332d`, muted `#727b72`, line `#d8ddd2`.
- Status: todo `#ebeee5`, doing `#e1ebf0`, done `#dcead6`, danger `#b8483e`.
- Radius: 2px for status bands, 3px for controls and surfaces.
- Elevation: borders and color contrast only; no drop shadows.
- Spacing: 4px base with intentional 10, 14, 22, 30, 42, and 56px group intervals.

## Layout

- Desktop target: 1440 x 900, 240px fixed rail, content origin at x=282px, working width 1064px.
- Mobile target: 390 x 844, 68px top bar, 22px side gutters, fixed 54px bottom navigation.
- Desktop and mobile use distinct information compositions where the Figma frames differ; mobile is not a scaled desktop layout.

## Components

- `AuthLayoutComponent`: shared split authentication shell.
- `PageHeaderComponent`: desktop and mobile page hierarchy with projected action.
- `FormFieldComponent`: labels, responsive label variants, controls, and textual errors.
- `TaskRowComponent`: canonical desktop task row.
- `TaskCardComponent`: canonical board card.
- `StatusBadgeComponent`: canonical task status token.
- `ConfirmDialogComponent`: destructive confirmation with focus trap, Escape handling, and focus restoration.

## Motion And Access

- Native Angular 20 `animate.enter`/`animate.leave` with CSS transform and opacity only.
- Page entry: 260ms ease-out; dialog entry: 220ms ease-out; overlay exit: 140ms ease-in.
- `prefers-reduced-motion` reduces all animation and transition durations.
- Visible focus, semantic controls, persistent labels, textual validation, keyboard-safe modal.

## Audit

- No gradients, glass effects, diffuse shadows, inflated cards, or decorative motion.
- Desktop and mobile frames follow Figma hierarchy, dimensions, palette, type, and status language.
- Changelog 2026-10-29: aligned Angular UI with Figma node `7:2`; extracted shared primitives and responsive compositions.
