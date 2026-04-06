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
- `docker-compose` inicial con una base PostgreSQL por servicio
- CI básica

Todavía **no** implementa persistencia ni lógica completa de negocio. Eso viene en los próximos pasos.

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

## Próximos pasos recomendados

1. Modelar entidades y estados de V1
2. Implementar persistencia PostgreSQL en `directory-service`
3. Implementar persistencia PostgreSQL en `appointments-service`
4. Agregar validaciones de negocio
5. Completar OpenAPI con request/response reales
6. Agregar tests por caso de uso
