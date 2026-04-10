import type { ReactNode } from 'react';
import type { ShellNavItem, ShellSurfaceIntro, SurfaceId } from '../auth/actorCapabilities';

export type AppShellBadge = {
  label: string;
  tone?: 'neutral' | 'info' | 'success' | 'error';
};

export type AppShellAccountSummary = {
  email: string;
  role: string;
  sessionExpiryLabel: string;
};

export type AppShellHeader = {
  productName: string;
  workspaceName: string;
  workspaceDescription: string;
  logo?: {
    src: string;
    alt: string;
  };
  badges?: AppShellBadge[];
};

export type AppShellSidebar = {
  eyebrow: string;
  title: string;
  description: string;
};

export type AppShellPageIntro = ShellSurfaceIntro & {
  badges?: AppShellBadge[];
  actions?: ReactNode;
};

export type AppShellBodySlot = {
  ariaLabel?: string;
  children: ReactNode;
};

export type AppShellProps = {
  header: AppShellHeader;
  account: AppShellAccountSummary;
  activeSurface: SurfaceId;
  sidebar: AppShellSidebar;
  pageIntro: AppShellPageIntro;
  body: AppShellBodySlot;
  navItems: ShellNavItem[];
  onLogout: () => void;
  onSelectSurface: (surfaceId: SurfaceId) => void;
};
