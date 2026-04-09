import { useEffect, useMemo, useState } from 'react';
import { deriveActorCapabilities, type SurfaceId } from './auth/actorCapabilities';
import { useAuthSession } from './auth/useAuthSession';
import { LoginScreen } from './features/auth/LoginScreen';
import { DirectoryDemo } from './features/directory/DirectoryDemo';
import { PatientsWorkspace } from './features/patients/PatientsWorkspace';
import { ScheduleDemo } from './features/schedule/ScheduleDemo';

type SurfaceDefinition = {
  id: SurfaceId;
  label: string;
  eyebrow: string;
  description: string;
};

export default function App() {
  const auth = useAuthSession();
  const capabilities = useMemo(
    () => (auth.user ? deriveActorCapabilities(auth.user) : null),
    [auth.user],
  );
  const availableSurfaces = useMemo(
    () => (capabilities ? capabilities.visibleSurfaces.map((surface) => getSurfaceDefinition(surface, capabilities)) : []),
    [capabilities],
  );
  const [activeSurface, setActiveSurface] = useState<SurfaceId>('agenda');

  useEffect(() => {
    if (!capabilities) {
      setActiveSurface('agenda');
      return;
    }

    if (!capabilities.visibleSurfaces.includes(activeSurface)) {
      setActiveSurface(capabilities.defaultSurface);
    }
  }, [activeSurface, capabilities]);

  const activeSurfaceDefinition = availableSurfaces.find((surface) => surface.id === activeSurface) ?? availableSurfaces[0];

  if (auth.status === 'loading') {
    return (
      <main className="page page-centered">
        <div className="shell auth-shell">
          <section className="card stack">
            <div className="hero-kicker">Autenticación</div>
            <h1>Restaurando sesión...</h1>
            <p>Estamos validando el token guardado con el backend nuevo.</p>
          </section>
        </div>
      </main>
    );
  }

  if (auth.status !== 'authenticated' || !auth.user) {
    return <LoginScreen errorMessage={auth.errorMessage} isSubmitting={auth.isSubmitting} onLogin={auth.login} />;
  }

  return (
    <main className="page">
      <div className="shell app-shell stack">
        <header className="hero hero-product card">
          <div className="hero-kicker">Demo frontend · sesión activa</div>
          <div className="hero-copy stack-tight">
            <h1>Clinic platform demos</h1>
            <p>
              Sprint 1 V2 con login mínimo, sesión simple en frontend y superficies filtradas por rol sin meter
              complejidad extra.
            </p>
          </div>

          <div className="hero-summary-grid">
            <article className="summary-tile">
              <span className="summary-label">Superficie activa</span>
              <strong>{activeSurfaceDefinition?.label ?? 'Agenda'}</strong>
              <small>{activeSurfaceDefinition?.description ?? 'Cada vista conserva su propio foco.'}</small>
            </article>
            <article className="summary-tile">
              <span className="summary-label">Sesión</span>
              <strong>{auth.user.email}</strong>
              <small>Rol: {auth.user.role}</small>
            </article>
          </div>

          <div className="toolbar shell-toolbar">
            <span className="badge neutral">Expira: {formatSessionExpiry(auth.expiresAt)}</span>
            {auth.user.role === 'admin' ? <span className="badge info">Admin: foco en setup y configuración</span> : null}
            <button className="button button-secondary" type="button" onClick={auth.logout}>
              Cerrar sesión
            </button>
          </div>
        </header>

        <section className="surface-switcher card stack-tight" aria-label="Selector de demo">
          <div className="surface-switcher-header">
            <div>
              <h2>Elegí la superficie</h2>
              <p>Mostramos solo lo mínimo para el rol autenticado, sin router ni shell gigante.</p>
            </div>
            <span className="badge neutral">{availableSurfaces.length} superficie(s) habilitada(s)</span>
          </div>

          <div className="surface-tabs" role="tablist" aria-label="Superficies demo">
            {availableSurfaces.map((surface) => (
              <button
                key={surface.id}
                type="button"
                role="tab"
                aria-selected={activeSurface === surface.id}
                className={`surface-tab${activeSurface === surface.id ? ' active' : ''}`}
                onClick={() => setActiveSurface(surface.id)}
              >
                <span className="surface-tab-eyebrow">{surface.eyebrow}</span>
                <strong>{surface.label}</strong>
                <small>{surface.description}</small>
              </button>
            ))}
          </div>
        </section>

        {activeSurface === 'agenda' && capabilities ? (
          <ScheduleDemo agendaMode={capabilities.agendaMode} onSessionInvalid={auth.logout} />
        ) : null}
        {activeSurface === 'directory' && capabilities ? (
          <DirectoryDemo directoryMode={capabilities.directoryMode} onSessionInvalid={auth.logout} />
        ) : null}
        {activeSurface === 'patients' && capabilities ? (
          <PatientsWorkspace patientsMode={capabilities.patientsMode} onSessionInvalid={auth.logout} />
        ) : null}
      </div>
    </main>
  );
}

function getSurfaceDefinition(surfaceId: SurfaceId, capabilities: NonNullable<ReturnType<typeof deriveActorCapabilities>>): SurfaceDefinition {
  if (surfaceId === 'agenda') {
    return capabilities.agendaMode.kind === 'doctor-own'
      ? { id: 'agenda', label: 'Mi agenda', eyebrow: 'Atención diaria', description: 'Vista enfocada en tu agenda profesional.' }
      : { id: 'agenda', label: 'Agenda', eyebrow: 'Operación diaria', description: 'Turnos, slots y gestión operativa.' };
  }

  if (surfaceId === 'patients') {
    return capabilities.patientsMode.kind === 'secretary-operational'
      ? {
          id: 'patients',
          label: 'Pacientes',
          eyebrow: 'Flujo operativo',
          description: 'Búsqueda y selección para tareas administrativas y agenda.',
        }
      : {
          id: 'patients',
          label: 'Pacientes',
          eyebrow: 'Atención clínica',
          description: 'Resumen clínico mínimo y encounters del paciente.',
        };
  }

  return {
    id: 'directory',
    label: 'Directorio',
    eyebrow: 'Setup base',
    description: 'Alta base de pacientes y profesionales.',
  };
}

function formatSessionExpiry(expiresAt: string | null) {
  if (!expiresAt) {
    return 'sin dato';
  }

  const parsedDate = new Date(expiresAt);
  if (Number.isNaN(parsedDate.getTime())) {
    return expiresAt;
  }

  return parsedDate.toLocaleString('es-AR', {
    dateStyle: 'short',
    timeStyle: 'short',
  });
}
