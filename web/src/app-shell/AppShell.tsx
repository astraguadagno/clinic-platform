import { Badge } from './AppShell.primitives';
import type { AppShellProps } from './AppShell.types';

export function AppShell({
  header,
  account,
  activeSurface,
  body,
  pageIntro,
  navItems,
  onLogout,
  onSelectSurface,
}: AppShellProps) {
  const activeNavItem = navItems.find((item) => item.id === activeSurface) ?? navItems[0];
  const surfaceToneClass = `app-shell-surface-${activeSurface}`;
  const productWords = header.productName.trim().split(/\s+/);
  const brandLead = productWords[0] ?? header.productName;
  const brandTail = productWords.slice(1).join(' ');
  const brandInitials = productWords
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? '')
    .join('');

  return (
    <main className={`page page-app-shell ${surfaceToneClass}`}>
      <div className="app-shell-layout">
        <aside className="app-shell-rail" aria-label="Navegación principal">
          <div className="app-shell-rail-brand-block stack">
            <div className="app-shell-brand">
              {header.logo ? (
                <span className="app-shell-brand-mark app-shell-brand-mark-image-wrap">
                  <img className="app-shell-brand-image" src={header.logo.src} alt={header.logo.alt} />
                </span>
              ) : (
                <span className="app-shell-brand-mark app-shell-brand-mark-fallback" aria-hidden="true">
                  {brandInitials}
                </span>
              )}
              <div className="stack-tight app-shell-brand-copy">
                <span className="hero-kicker">{header.workspaceName}</span>
                <strong className="app-shell-brand-wordmark">
                  <span>{brandLead}</span>
                  {brandTail ? <span>{brandTail}</span> : null}
                </strong>
              </div>
            </div>
          </div>

          <nav className="app-shell-nav" aria-label="Áreas disponibles">
            <ul className="app-shell-nav-list" role="list">
              {navItems.map((surface) => (
                <li key={surface.id}>
                  <button
                    type="button"
                    aria-pressed={activeSurface === surface.id}
                    className={`app-shell-nav-item${activeSurface === surface.id ? ' active' : ''}`}
                    onClick={() => onSelectSurface(surface.id)}
                  >
                    <span className="app-shell-nav-item-marker" aria-hidden="true" />
                    <span className="app-shell-nav-item-copy">
                      <span className="surface-tab-eyebrow">{surface.eyebrow}</span>
                      <strong>{surface.label}</strong>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </nav>

          <footer className="app-shell-rail-footer stack-tight" aria-label="Cuenta">
            <div className="app-shell-rail-account stack-tight">
              <span className="summary-label">Cuenta activa</span>
              <strong>{account.email}</strong>
              <small>{account.role}</small>
            </div>

            <button className="button button-secondary app-shell-rail-logout" type="button" onClick={onLogout}>
              Cerrar sesión
            </button>
          </footer>
        </aside>

        <div className="app-shell-column">
          <header className="app-shell-topbar">
            <div className="app-shell-topbar-context stack-tight">
              <span className="summary-label">{header.workspaceName}</span>
              <strong>{activeNavItem?.label ?? pageIntro.title}</strong>
            </div>

            <div className="app-shell-topbar-meta">
              <div className="app-shell-topbar-account stack-tight" aria-label="Sesión activa">
                <span className="summary-label">Sesión activa</span>
                <strong>{account.email}</strong>
              </div>

              <div className="app-shell-topbar-badges">
                <Badge>Expira: {account.sessionExpiryLabel}</Badge>
              </div>
            </div>
          </header>

          <section className="app-shell-page-intro">
            <div className="app-shell-page-intro-copy stack-tight">
              <span className="hero-kicker">{pageIntro.eyebrow}</span>
              <h1>{pageIntro.title}</h1>
              <p>{pageIntro.description}</p>
            </div>
          </section>

          <section className="app-shell-stage" aria-label={body.ariaLabel ?? `Contenido de ${pageIntro.title}`}>
            <div className="app-shell-stage-frame">
              <div className="app-shell-body-stage">{body.children}</div>
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
