# Change Proposal — weekly-schedule-template-flow-ui

## Why
Clinic Platform ya modela templates semanales, vigencia (`effective_from`) y blocks a nivel backend, pero todavía no tiene una definición SDD explícita del flujo y la UI para que secretaría o doctor fijen la agenda semanal de un profesional sin ambigüedad.

Hoy el riesgo no es técnico solamente; es de producto y operación:
- no está definida la superficie para crear o editar un template semanal
- no está resuelto cómo se previsualiza el impacto futuro antes de guardar
- no está claro cómo difieren los permisos y affordances entre `Secretary` y `Doctor`
- no está aterrizada la visibilidad de conflictos con consultas/turnos futuros ya reservados

Sin ese plano funcional, la implementación puede derivar en una UI que mezcle edición de agenda con booking diario y termine siendo difícil de usar y mantener.

## What Changes
- Definir el flujo de creación/edición de agenda semanal por profesional.
- Definir la separación de experiencia entre secretaría multi-médico y doctor sobre agenda propia.
- Definir la UI mínima para template semanal, vigencia, preview y conflictos.
- Alinear el lenguaje de la interfaz con el modelo vigente de templates/versiones sin forzar todavía la implementación completa.

## Affected Layers
- Frontend product/UX for weekly schedule editing
- Frontend component architecture around schedule editing surfaces
- Backend contract assumptions consumed by the future UI (`/schedules`, `/schedules/versions`, `/agenda/week`)
- Operational rules shared with related backlog items about conflicts and multi-doctor secretariat workflows

## Relationship to Existing Changes
- `agenda-week-ui-migration` cubre la migración técnica de lectura semanal y adapter.
- Este change cubre la definición funcional/UX de la pantalla y el flujo para fijar agendas semanales.
- Ambos changes son complementarios, no duplicados.

## Risks
- Mezclar edición de template con operación diaria de agenda en la misma superficie.
- Diseñar una UI que no explique con claridad qué cambia desde `effective_from`.
- Hacer una experiencia única para `Secretary` y `Doctor` cuando sus necesidades operativas no son iguales.

## Rollback Plan
- Como este change es principalmente de definición funcional, el rollback consiste en no adoptar la propuesta y mantener la UI actual mientras se reformula el flow.
- No requiere eliminar contratos backend ni migraciones existentes.

## Success Criteria
- Existe una definición explícita de flujo para crear/editar template semanal por profesional.
- Está resuelta la diferencia de experiencia entre `Secretary` y `Doctor`.
- Preview y conflictos futuros están incluidos antes de confirmar cambios.
- La futura implementación puede descomponerse en componentes y tareas concretas sin ambigüedad de producto.
