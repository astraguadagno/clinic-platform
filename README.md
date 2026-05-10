# clinic-platform

Monorepo para una plataforma de consultorio en evolución. Hoy combina servicios Go con PostgreSQL y un frontend React/Vite orientado al workspace clínico: agenda, pacientes, directorio y sesiones autenticadas por rol.

## Estructura

```text
clinic-platform/
  deploy/
    docker-compose.yml          # backend local + PostgreSQL por servicio
  services/
    directory-service/          # pacientes, profesionales, auth y núcleo clínico
    appointments-service/       # agenda, turnos, consultas y solicitudes
  web/                          # frontend React/Vite + Vitest/Testing Library
  openspec/                     # specs y cambios guiados por SDD
```

## Backend local

Prerequisito: Docker Desktop o engine compatible con `docker compose`.

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Servicios principales:

- `directory-service` → `http://localhost:8081`
- `appointments-service` → `http://localhost:8082`
- `directory-db` → `localhost:5433`
- `appointments-db` → `localhost:5434`

El compose levanta una base PostgreSQL por servicio, aplica el bootstrap inicial y ejecuta migradores locales para el esquema actual.

Para resetear datos locales:

```bash
docker compose -f deploy/docker-compose.yml down -v
```

> `down -v` borra los volúmenes locales de PostgreSQL.

## Frontend local

```bash
cd web
npm install
npm run dev
```

Scripts disponibles en `web/`:

- `npm run dev` — Vite dev server
- `npm test` — Vitest en modo run
- `npm run typecheck` — TypeScript sin emitir archivos
- `npm run preview` — preview local de Vite

## Verificación rápida

Con el backend levantado:

```bash
curl http://localhost:8081/health
curl http://localhost:8082/health
```

Para cambios de frontend, preferí verificación dirigida (`npm test`, tests específicos o `npm run typecheck` según el cambio). La convención del proyecto es no correr builds salvo pedido explícito.

## Dirección actual

- Mantener una separación clara entre backend Go, contratos HTTP/OpenAPI y adaptación frontend.
- En pacientes, preservar la diferencia entre `clinical_history` editable y `clinical_notes` como notas/eventos clínicos.
- Evolucionar la UI hacia un shell autenticado contextual: navegación por áreas, workspace de pacientes ficha-first, historia/notas clínicas en segundo plano y agenda operativa.
