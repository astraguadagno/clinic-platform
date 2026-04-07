import { useEffect, useMemo, useState } from 'react';
import { useAuthSession } from './auth/useAuthSession';
import { LoginScreen } from './features/auth/LoginScreen';
import { DirectoryDemo } from './features/directory/DirectoryDemo';
import { PatientsWorkspace } from './features/patients/PatientsWorkspace';
import { ScheduleDemo } from './features/schedule/ScheduleDemo';

type DemoSurface = 'agenda' | 'directory' | 'patients';

type SurfaceDefinition = {
  id: DemoSurface;
  label: string;
  eyebrow: string;
  description: string;
};

const SURFACES_BY_ROLE: Record<string, SurfaceDefinition[]> = {
  secretary: [
    { id: 'agenda', label: 'Agenda', eyebrow: 'Operación diaria', description: 'Turnos, slots y gestión básica.' },
    { id: 'directory', label: 'Directorio', eyebrow: 'Carga base', description: 'Pacientes y profesionales.' },
  ],
  doctor: [
    { id: 'agenda', label: 'Mi agenda', eyebrow: 'Atención diaria', description: 'Vista simple para la jornada.' },
    { id: 'patients', label: 'Pacientes', eyebrow: 'Placeholder', description: 'Superficie mínima para Sprint 1.' },
  ],
  admin: [
    { id: 'agenda', label: 'Agenda', eyebrow: 'Operación diaria', description: 'Turnos, slots y gestión básica.' },
    { id: 'directory', label: 'Directorio', eyebrow: 'Carga base', description: 'Pacientes y profesionales.' },
  ],
};

const DEFAULT_SURFACES: SurfaceDefinition[] = [
  { id: 'agenda', label: 'Agenda', eyebrow: 'Operación diaria', description: 'Acceso mínimo mientras resolvemos el rol.' },
];

export default function App() {
  const auth = useAuthSession();
  const availableSurfaces = useMemo(() => {
    const role = auth.user?.role;
    return SURFACES_BY_ROLE[role ?? ''] ?? DEFAULT_SURFACES;
  }, [auth.user?.role]);
  const [activeSurface, setActiveSurface] = useState<DemoSurface>('agenda');

  useEffect(() => {
    if (!availableSurfaces.some((surface) => surface.id === activeSurface)) {
      setActiveSurface(availableSurfaces[0]?.id ?? 'agenda');
    }
  }, [activeSurface, availableSurfaces]);

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
            {auth.user.role === 'admin' ? <span className="badge info">Admin: acceso amplio, shell mínimo por ahora</span> : null}
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

        {activeSurface === 'agenda' ? <ScheduleDemo /> : null}
        {activeSurface === 'directory' ? <DirectoryDemo /> : null}
        {activeSurface === 'patients' ? <PatientsWorkspace currentUser={auth.user} /> : null}
      </div>
    </main>
  );
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
