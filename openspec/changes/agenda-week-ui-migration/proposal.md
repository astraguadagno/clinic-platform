# Change Proposal — agenda-week-ui-migration

## Why
`appointments-service` ya expone el modelo nuevo de agenda semanal compuesto (`/agenda/week`) y el dominio operativo evolucionó hacia `consultations`, templates y blocks. Sin embargo, la UI principal (`ScheduleDemo`) todavía recompone la semana con `listSlots()` + `listAppointments()` y sigue centrada en `appointments` legacy.

Eso deja al producto en una situación híbrida:
- el backend piensa en `consultations`
- la pantalla principal piensa en `appointments`
- el flujo visual depende de 10 requests por semana en vez de una proyección compuesta

La migración propuesta cierra esa brecha sin romper compatibilidad de golpe.

## What Changes
- Introducir un adapter explícito `WeekAgenda -> board model` en frontend.
- Migrar `ScheduleDemo` para leer `/agenda/week` como fuente única de lectura semanal.
- Adaptar el tablero para renderizar `consultations` sin mezclar transformación de dominio con JSX.
- Mantener temporalmente las operaciones legacy (`/appointments`) como compatibilidad mientras se completa la migración de escritura.
- Reescribir los tests del tablero para validar el contrato nuevo (`fetchWeekAgenda`) en vez del armado legacy por día.

## Affected Layers
- Frontend adapter/model: `web/src/features/schedule/*`
- Frontend API client: `web/src/api/appointments.ts`
- Frontend tests: `web/src/features/schedule/ScheduleDemo.test.tsx`, `web/src/api/appointments.test.ts`
- Backend contract relied upon: `/agenda/week`, `/consultations`, `/appointments` compatibility

## Compatibility Strategy
- `/agenda/week` pasa a ser la fuente primaria de lectura para la agenda UI.
- `/appointments` permanece como superficie de compatibilidad temporal hasta migrar escritura/cancelación.
- La UI seguirá mostrando una representación visual simple de estados (`scheduled` como reservado) en la primera iteración para evitar mezclar migración conceptual con rediseño UX completo.

## Risks
- Aumentar complejidad accidental en `ScheduleDemo` si no se extrae el adapter.
- Inconsistencias visuales si se mezclan en la misma pantalla datos legacy y compuestos.
- Mantener demasiado tiempo dos modelos conceptuales activos (`appointments` vs `consultations`).

## Rollback Plan
- Si la lectura nueva falla, el cambio se puede revertir devolviendo `ScheduleDemo` a `listOperationalWeek()` mientras se conserva intacto el contrato backend.
- No se eliminan endpoints legacy en esta fase.

## Success Criteria
- La agenda principal usa una sola lectura semanal (`/agenda/week`).
- La transformación de dominio vive fuera del componente React.
- Los tests del tablero validan el contrato nuevo.
- La UI deja de depender del armado legacy semanal por requests diarios.
