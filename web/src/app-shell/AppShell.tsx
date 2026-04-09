import type { AppShellProps } from './AppShell.types';

export function AppShell({
  account,
  activeSurface,
  activeSurfaceCountLabel,
  children,
  intro,
  navItems,
  onLogout,
  onSelectSurface,
}: AppShellProps) {
  const activeNavItem = navItems.find((item) => item.id === activeSurface) ?? navItems[0];

  return (
    <main className="page">
      <div className="shell app-shell-frame stack">
        <header className="card app-shell-header stack">
          <div className="app-shell-header-main">
            <div className="stack-tight hero-copy">
              <div className="hero-kicker">Clinic Platform · sesión activa</div>
              <h1>Panel de trabajo de la clínica</h1>
              <p>
                Entrá con tu perfil y seguí la operación diaria desde los espacios habilitados para ese rol, sin cambiar
                el flujo actual.
              </p>
            </div>

            <div className="hero-summary-grid">
              <article className="summary-tile">
                <span className="summary-label">Espacio actual</span>
                <strong>{activeNavItem?.label ?? intro.title}</strong>
                <small>{activeNavItem?.description ?? intro.description}</small>
              </article>
              <article className="summary-tile">
                <span className="summary-label">Cuenta</span>
                <strong>{account.email}</strong>
                <small>Perfil: {account.role}</small>
              </article>
            </div>
          </div>

          <div className="toolbar shell-toolbar">
            <span className="badge neutral">Expira: {account.sessionExpiryLabel}</span>
            {account.isAdmin ? <span className="badge info">Admin: mantenimiento y configuración base</span> : null}
            <button className="button button-secondary" type="button" onClick={onLogout}>
              Cerrar sesión
            </button>
          </div>
        </header>

        <section className="card app-shell-nav stack-tight" aria-label="Selector de espacios">
          <div className="app-shell-nav-header">
            <div>
              <h2>Tu espacio de trabajo</h2>
              <p>Mostramos solamente las áreas disponibles para este perfil, con el mismo acceso y permisos de hoy.</p>
            </div>
            <span className="badge neutral">{activeSurfaceCountLabel}</span>
          </div>

          <div className="surface-tabs" role="tablist" aria-label="Áreas disponibles">
            {navItems.map((surface) => (
              <button
                key={surface.id}
                type="button"
                role="tab"
                aria-selected={activeSurface === surface.id}
                className={`surface-tab${activeSurface === surface.id ? ' active' : ''}`}
                onClick={() => onSelectSurface(surface.id)}
              >
                <span className="surface-tab-eyebrow">{surface.eyebrow}</span>
                <strong>{surface.label}</strong>
                <small>{surface.description}</small>
              </button>
            ))}
          </div>
        </section>

        <section className="card app-shell-surface-frame stack-tight">
          <div className="stack-tight app-shell-surface-intro">
            <span className="hero-kicker">{intro.eyebrow}</span>
            <h2>{intro.title}</h2>
            <p>{intro.description}</p>
          </div>
        </section>

        <div className="app-shell-surface-content">{children}</div>
      </div>
    </main>
  );
}
