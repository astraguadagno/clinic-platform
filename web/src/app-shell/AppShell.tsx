import type { ShellNavItem, SurfaceId } from '../auth/actorCapabilities';
import type { AppShellAccountSummary, AppShellHeader, AppShellProps } from './AppShell.types';

type BrandIdentity = {
  lead: string;
  tail: string;
  initials: string;
};

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
  const brand = buildBrandIdentity(header.productName);

  return (
    <main className={`page page-app-shell min-h-screen bg-clinic-background text-clinic-text ${surfaceToneClass}`}>
      <div className="clinic-shell-layout min-h-screen md:pl-[280px]">
        <aside
          className="clinic-shell-sidebar border-clinic-border bg-clinic-surface/95 z-30 flex w-full flex-col border-b px-5 py-6 shadow-sm md:fixed md:inset-y-0 md:left-0 md:w-[280px] md:border-b-0 md:border-r"
          aria-label="Navegación principal"
        >
          <Brand header={header} brand={brand} />
          <PrimaryAction />
          <SidebarNav activeSurface={activeSurface} navItems={navItems} onSelectSurface={onSelectSurface} />
          <SidebarFooter onLogout={onLogout} />
        </aside>

        <div className="clinic-shell-column min-h-screen">
          <Topbar account={account} title={activeNavItem?.label ?? pageIntro.title} workspaceName={header.workspaceName} />

          <section className="clinic-shell-stage px-5 py-6 md:px-8 md:pb-10 md:pt-28" aria-label={body.ariaLabel ?? `Contenido de ${pageIntro.title}`}>
            <div className="app-shell-body-stage">{body.children}</div>
          </section>
        </div>
      </div>
    </main>
  );
}

function Brand({ header, brand }: { header: AppShellHeader; brand: BrandIdentity }) {
  return (
    <div className="flex items-center gap-3 border-b border-clinic-border/70 pb-5">
      <div className="app-shell-brand min-w-0">
        {header.logo ? (
          <span className="app-shell-brand-mark-image-wrap flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-clinic-border">
            <img className="app-shell-brand-image" src={header.logo.src} alt={header.logo.alt} />
          </span>
        ) : (
          <span
            className="app-shell-brand-mark-fallback flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-clinic-accent text-sm font-black tracking-[0.14em] text-clinic-primaryDark shadow-sm"
            aria-hidden="true"
          >
            {brand.initials}
          </span>
        )}
        <div className="min-w-0">
          <span className="block truncate text-[0.68rem] font-extrabold uppercase tracking-[0.14em] text-clinic-muted">{header.workspaceName}</span>
          <strong className="mt-1 flex flex-wrap gap-x-1 text-xl font-black tracking-tight text-clinic-primaryDark">
            <span>{brand.lead}</span>
            {brand.tail ? <span>{brand.tail}</span> : null}
          </strong>
        </div>
      </div>
    </div>
  );
}

function PrimaryAction() {
  return (
    <button
      type="button"
      className="clinic-shell-primary-action mt-6 inline-flex w-full items-center justify-center rounded-2xl bg-clinic-primary px-4 py-3 text-sm font-extrabold text-white shadow-clinic-soft transition hover:bg-clinic-primaryDark focus:outline-none focus:ring-2 focus:ring-clinic-primary focus:ring-offset-2"
      aria-label="Crear nuevo turno"
    >
      + Nuevo Turno
    </button>
  );
}

function SidebarNav({
  activeSurface,
  navItems,
  onSelectSurface,
}: {
  activeSurface: SurfaceId;
  navItems: ShellNavItem[];
  onSelectSurface: (surfaceId: SurfaceId) => void;
}) {
  return (
    <nav className="mt-7 flex-1" aria-label="Áreas disponibles">
      <ul className="clinic-shell-nav-list space-y-2" role="list">
        {navItems.map((surface) => (
          <li key={surface.id}>
            <NavButton surface={surface} isActive={activeSurface === surface.id} onSelectSurface={onSelectSurface} />
          </li>
        ))}
      </ul>
    </nav>
  );
}

function NavButton({
  surface,
  isActive,
  onSelectSurface,
}: {
  surface: ShellNavItem;
  isActive: boolean;
  onSelectSurface: (surfaceId: SurfaceId) => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={isActive}
      className={`clinic-shell-nav-button group flex w-full items-center gap-3 rounded-2xl border px-4 py-3 text-left transition ${navButtonStateClass(isActive)}`}
      onClick={() => onSelectSurface(surface.id)}
    >
      <span className={`clinic-shell-nav-marker h-2.5 w-2.5 rounded-full ${navMarkerStateClass(isActive)}`} aria-hidden="true" />
      <span className="min-w-0">
        <span className="block text-[0.67rem] font-black uppercase tracking-[0.12em] text-current/70">{surface.eyebrow}</span>
        <strong className="block truncate text-sm font-extrabold">{surface.label}</strong>
      </span>
    </button>
  );
}

function SidebarFooter({ onLogout }: { onLogout: () => void }) {
  return (
    <footer className="clinic-shell-footer mt-6 space-y-3 border-t border-clinic-border/70 pt-5" aria-label="Cuenta">
      <button
        type="button"
        className="clinic-shell-help w-full rounded-2xl px-3 py-2 text-left text-sm font-bold text-clinic-muted transition hover:bg-[#fff8eb]"
      >
        Ayuda
      </button>

      <button
        className="clinic-shell-logout w-full rounded-2xl border border-clinic-border bg-white px-3 py-2 text-left text-sm font-extrabold text-clinic-primary transition hover:bg-[#fff8eb]"
        type="button"
        onClick={onLogout}
      >
        Cerrar sesión
      </button>
    </footer>
  );
}

function Topbar({ account, title, workspaceName }: { account: AppShellAccountSummary; title: string; workspaceName: string }) {
  return (
    <header className="clinic-shell-topbar border-clinic-border bg-clinic-surface/95 z-20 border-b px-5 py-4 shadow-sm backdrop-blur md:fixed md:left-[280px] md:right-0 md:top-0 md:h-20 md:px-8">
      <div className="clinic-shell-topbar-inner flex flex-col gap-4 md:h-full md:flex-row md:items-center md:justify-between">
        <div className="clinic-shell-topbar-title min-w-0">
          <span className="block text-[0.68rem] font-black uppercase tracking-[0.14em] text-clinic-muted">{workspaceName}</span>
          <h1 className="truncate text-2xl font-black tracking-tight text-clinic-text">{title}</h1>
        </div>

        <PatientSearch />

        <div className="clinic-shell-actions flex shrink-0 items-center gap-3">
          <NotificationsButton />
          <AccountSummary account={account} />
        </div>
      </div>
    </header>
  );
}

function PatientSearch() {
  return (
    <label className="clinic-shell-search relative w-full max-w-xl md:flex-1" aria-label="Buscar paciente">
      <span className="clinic-shell-search-icon pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-clinic-muted" aria-hidden="true">
        ⌕
      </span>
      <input
        type="search"
        className="clinic-shell-search-input h-11 w-full rounded-2xl border border-clinic-border bg-[#fdfaf5] pl-10 pr-4 text-sm text-clinic-text outline-none transition placeholder:text-clinic-muted focus:border-clinic-primary focus:bg-white focus:ring-2 focus:ring-clinic-primary/15"
        placeholder="Buscar paciente por nombre o DNI..."
        aria-label="Buscar paciente por nombre o DNI"
      />
    </label>
  );
}

function NotificationsButton() {
  return (
    <button
      type="button"
      className="clinic-shell-notification relative flex h-11 w-11 items-center justify-center rounded-2xl border border-clinic-border bg-white text-clinic-primary shadow-sm transition hover:bg-[#fff8eb]"
      aria-label="Abrir notificaciones"
    >
      <span aria-hidden="true">●</span>
      <span className="clinic-shell-notification-dot absolute right-2.5 top-2.5 h-2 w-2 rounded-full bg-clinic-primary" aria-hidden="true" />
    </button>
  );
}

function AccountSummary({ account }: { account: AppShellAccountSummary }) {
  return (
    <div
      className="clinic-shell-account flex items-center gap-3 rounded-2xl border border-clinic-border bg-white px-3 py-2 shadow-sm"
      aria-label="Sesión activa"
    >
      <span className="clinic-shell-account-avatar flex h-10 w-10 items-center justify-center rounded-xl bg-clinic-primary text-sm font-black text-white" aria-hidden="true">
        {buildAccountInitials(account.email)}
      </span>
      <div className="clinic-shell-account-summary hidden min-w-0 sm:block">
        <strong className="block max-w-[180px] truncate text-sm font-extrabold text-clinic-text">{account.email}</strong>
        <span className="block text-xs font-bold capitalize text-clinic-muted">{account.role}</span>
        <span className="sr-only">Expira: {account.sessionExpiryLabel}</span>
      </div>
    </div>
  );
}

function navButtonStateClass(isActive: boolean) {
  return isActive
    ? 'clinic-shell-nav-button-active border-clinic-primary/20 bg-clinic-accent/75 text-clinic-primaryDark shadow-sm'
    : 'clinic-shell-nav-button-idle border-transparent text-clinic-muted hover:border-clinic-border hover:bg-[#fff8eb] hover:text-clinic-text';
}

function navMarkerStateClass(isActive: boolean) {
  return isActive ? 'bg-clinic-primary' : 'bg-clinic-border group-hover:bg-clinic-accent';
}

function buildBrandIdentity(productName: string): BrandIdentity {
  const productWords = productName.trim().split(/\s+/).filter(Boolean);
  const lead = productWords[0] ?? productName;
  const tail = productWords.slice(1).join(' ');
  const initials = productWords
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? '')
    .join('');

  return { lead, tail, initials };
}

function buildAccountInitials(email: string) {
  const namePart = email.split('@')[0]?.trim() ?? '';
  const initials = namePart
    .split(/[._\-\s]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');

  return initials || 'U';
}
