# clinic-platform

V1 inicial de una plataforma para consultorio, pensada para crecer de menos a más con foco en:

- backend en Go
- arquitectura basada en microservicios mínimos
- contratos HTTP claros con OpenAPI
- práctica de cloud/devops sin sobrecargar la primera iteración

## Servicios de la V1

- `directory-service`
  - pacientes
  - profesionales
- `appointments-service`
  - slots de disponibilidad
  - reservas de turnos
  - cancelaciones

## Objetivo de esta base inicial

Este primer scaffold deja:

- estructura del monorepo
- dos servicios HTTP mínimos
- endpoints de `health` e `info`
- rutas placeholder para la API de negocio
- OpenAPI base por servicio
- Dockerfiles por servicio
- `docker-compose` local con una base PostgreSQL por servicio e inicialización automática de esquema SQL
- CI básica

La V1 YA corre con persistencia real en PostgreSQL para ambos servicios. El alcance sigue siendo intencionalmente chico: APIs mínimas, esquemas iniciales y flujo local simple para operar y validar el entorno.

## Estructura

```text
clinic-platform/
  .github/workflows/
  deploy/
  services/
    directory-service/
    appointments-service/
```

## Servicios y puertos

- `directory-service` → `http://localhost:8081`
- `appointments-service` → `http://localhost:8082`
- `directory-db` → `localhost:5433`
- `appointments-db` → `localhost:5434`

## Arranque local correcto

Prerequisito: Docker Desktop o engine compatible con `docker compose`.

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Qué hace este arranque:

- levanta una DB PostgreSQL por servicio
- monta `001_init.sql` en `/docker-entrypoint-initdb.d/`
- inicializa el esquema actual en el primer arranque del volumen
- espera a que cada DB esté saludable antes de levantar sus servicios HTTP

> Importante: los scripts de inicialización corren solamente cuando el volumen de la base está vacío.
> En local, `appointments-service` bootstrappea directo el schema actual desde `001_init.sql` en vez de reconstruir la historia completa de migraciones sobre volúmenes existentes.

### Bootstrap local de appointments

Para mantener el entorno local simple y predecible:

- `001_init.sql` ya representa el esquema actual de `appointments-service`
- el compose local no intenta re-ejecutar `002` a `007` sobre volúmenes viejos
- si querés alinear tu entorno con el estado actual del schema, lo correcto es recrear el volumen local

El bootstrap local hoy cubre directamente:

- no solapamiento real de slots por profesional (`002`)
- templates de agenda versionados por profesional (`004`)
- bloqueos de agenda por fecha, rango o template (`005`)
- evolución de `appointments` a `consultations` con estados y metadatos operativos (`006`)
- rango horario propio de `consultations` para soportar consultas con o sin slot (`007`)

Tradeoff explícito:

- esto deja el compose local mucho más limpio
- pero asume que para adoptar el schema actual en development se puede resetear el volumen local cuando haga falta
- no reemplaza una estrategia formal de migraciones versionadas para staging/producción

## Reset del entorno local

Si necesitás recrear las bases desde cero y volver a bootstrappear el schema actual:

```bash
docker compose -f deploy/docker-compose.yml down -v
docker compose -f deploy/docker-compose.yml up --build
```

Usá `down -v` solo cuando quieras borrar los datos locales de PostgreSQL.

## Endpoints base

### Directory

- `GET /health`
- `GET /info`
- `POST /patients`
- `GET /patients`
- `GET /patients/{id}`
- `POST /professionals`
- `GET /professionals`
- `GET /professionals/{id}`

### Appointments

- `GET /health`
- `GET /info`
- `POST /slots/bulk`
- `GET /slots`
- `POST /appointments`
- `GET /appointments`
- `PATCH /appointments/{id}/cancel`

## Smoke checks mínimos

Con el stack levantado:

```bash
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8081/info
curl http://localhost:8082/info
```

Chequeos rápidos de persistencia:

```bash
curl -X POST http://localhost:8081/patients \
  -H 'Content-Type: application/json' \
  -d '{"first_name":"Ada","last_name":"Lovelace","document":"123","birth_date":"1990-10-10","phone":"555-0101"}'

curl http://localhost:8081/patients
```

Si querés inspeccionar las DB directamente:

```bash
docker compose -f deploy/docker-compose.yml exec directory-db psql -U directory -d directory
docker compose -f deploy/docker-compose.yml exec appointments-db psql -U appointments -d appointments
```

## Próximos pasos recomendados

1. Agregar readiness/healthchecks HTTP de aplicación si se quiere endurecer la dependencia entre servicios
2. Agregar datos seed opcionales para demo local sin mezclar esquema con fixtures
3. Completar validaciones de negocio y respuestas OpenAPI reales
4. Agregar tests de integración contra PostgreSQL por servicio
