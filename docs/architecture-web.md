# Web Architecture

**Part ID:** `web`  
**Path:** `apps/web/www/`  
**Stack:** Vite + React + TypeScript + Tailwind

## Summary

The web app is the public-facing website for MildStack.  
It is intentionally simple compared to `core/` and does not contain runtime emulation logic.

## Structure

- `src/components/`: shared UI and layout primitives
- `src/features/home/`: homepage composition and sections
- `src/lib/`: lightweight helper utilities
- `src/hooks/`: reusable client hooks

## Design System Pattern

- Tailwind CSS v4 + CSS variables
- Shared UI primitives in `components/ui`
- Motion-based interactions for key sections

## Runtime/Integration

- Browser SPA built with Vite
- No direct service orchestration responsibilities
- Deployment handled via GitHub Pages workflow

