import type { ReactNode } from 'react';
import type { ShellNavItem, ShellSurfaceIntro, SurfaceId } from '../auth/actorCapabilities';

export type AppShellAccountSummary = {
  email: string;
  role: string;
  sessionExpiryLabel: string;
  isAdmin: boolean;
};

export type AppShellProps = {
  account: AppShellAccountSummary;
  activeSurface: SurfaceId;
  activeSurfaceCountLabel: string;
  intro: ShellSurfaceIntro;
  navItems: ShellNavItem[];
  onLogout: () => void;
  onSelectSurface: (surfaceId: SurfaceId) => void;
  children: ReactNode;
};
