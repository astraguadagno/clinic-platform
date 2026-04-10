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

  const normalizedActiveSurface = capabilities?.visibleSurfaces.includes(activeSurface) ? activeSurface : capabilities?.defaultSurface;
  const activeSurfaceDefinition = availableSurfaces.find((surface) => surface.id === normalizedActiveSurface) ?? availableSurfaces[0];

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

  const renderActiveSurface = () => {
    if (!capabilities) {
      return null;
    }

    if (normalizedActiveSurface === 'agenda') {
      return <ScheduleDemo agendaMode={capabilities.agendaMode} onSessionInvalid={auth.logout} />;
    }

    if (normalizedActiveSurface === 'directory') {
      return <DirectoryDemo directoryMode={capabilities.directoryMode} onSessionInvalid={auth.logout} />;
    }

    if (normalizedActiveSurface === 'patients') {
      return <PatientsWorkspace patientsMode={capabilities.patientsMode} onSessionInvalid={auth.logout} />;
    }

    return null;
  };

  return (
      <AppShell
      header={{
        productName: 'Amicus',
        workspaceName: 'Centro operativo clínico',
        workspaceDescription: '',
      }}
      account={{
        email: auth.user.email,
        role: auth.user.role,
        sessionExpiryLabel: formatSessionExpiry(auth.expiresAt),
      }}
      activeSurface={normalizedActiveSurface ?? activeSurface}
      sidebar={{
        eyebrow: 'Espacios',
        title: 'Panel de trabajo',
        description: '',
      }}
      pageIntro={{
        ...(activeSurfaceDefinition?.intro ?? {
          eyebrow: 'Espacio',
          title: 'Agenda',
          description: 'Cada vista conserva su propio foco.',
        }),
      }}
      body={{
        ariaLabel: `Contenido de ${activeSurfaceDefinition?.intro.title ?? 'Agenda'}`,
        children: renderActiveSurface(),
      }}
      navItems={availableSurfaces.map((surface) => surface.navItem)}
      onLogout={auth.logout}
      onSelectSurface={setActiveSurface}
    />
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
