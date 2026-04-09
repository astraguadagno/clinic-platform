import { useEffect, useMemo, useState } from 'react';
import { AppShell } from './app-shell/AppShell';
import { deriveActorCapabilities, resolveShellSurfaceMetadata, type SurfaceId } from './auth/actorCapabilities';
import { useAuthSession } from './auth/useAuthSession';
import { LoginScreen } from './features/auth/LoginScreen';
import { DirectoryDemo } from './features/directory/DirectoryDemo';
import { PatientsWorkspace } from './features/patients/PatientsWorkspace';
import { ScheduleDemo } from './features/schedule/ScheduleDemo';

export default function App() {
  const auth = useAuthSession();
  const capabilities = useMemo(
    () => (auth.user ? deriveActorCapabilities(auth.user) : null),
    [auth.user],
  );
  const availableSurfaces = useMemo(
    () =>
      capabilities
        ? capabilities.visibleSurfaces.map((surface) => ({
            id: surface,
            ...resolveShellSurfaceMetadata(surface, capabilities),
          }))
        : [],
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
            <div className="hero-kicker">Acceso</div>
            <h1>Recuperando tu sesión...</h1>
            <p>Estamos validando tus credenciales para abrir el espacio de trabajo.</p>
          </section>
        </div>
      </main>
    );
  }

  if (auth.status !== 'authenticated' || !auth.user) {
    return <LoginScreen errorMessage={auth.errorMessage} isSubmitting={auth.isSubmitting} onLogin={auth.login} />;
  }

  return (
    <AppShell
      account={{
        email: auth.user.email,
        role: auth.user.role,
        sessionExpiryLabel: formatSessionExpiry(auth.expiresAt),
        isAdmin: auth.user.role === 'admin',
      }}
      activeSurface={activeSurface}
      activeSurfaceCountLabel={`${availableSurfaces.length} área(s) habilitada(s)`}
      intro={activeSurfaceDefinition?.intro ?? { eyebrow: 'Espacio', title: 'Agenda', description: 'Cada vista conserva su propio foco.' }}
      navItems={availableSurfaces.map((surface) => surface.navItem)}
      onLogout={auth.logout}
      onSelectSurface={setActiveSurface}
    >
      {activeSurface === 'agenda' && capabilities ? <ScheduleDemo agendaMode={capabilities.agendaMode} onSessionInvalid={auth.logout} /> : null}
      {activeSurface === 'directory' && capabilities ? (
        <DirectoryDemo directoryMode={capabilities.directoryMode} onSessionInvalid={auth.logout} />
      ) : null}
      {activeSurface === 'patients' && capabilities ? (
        <PatientsWorkspace patientsMode={capabilities.patientsMode} onSessionInvalid={auth.logout} />
      ) : null}
    </AppShell>
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
