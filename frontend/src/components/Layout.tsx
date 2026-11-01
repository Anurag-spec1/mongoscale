import { NavLink, Navigate, Outlet, useLocation } from 'react-router-dom';
import { isAdmin, useAuthStore } from '../store/auth';
import { api } from '../services/api';
import type { MetricsResponse } from '../types';
import { usePolling } from '../hooks/usePolling';
import { Icon } from './icons';
import { classNames } from '../utils/format';

const navigation = [
  { to: '/dashboard', label: 'Dashboard', icon: 'dashboard' as const },
  { to: '/architecture', label: 'Architecture', icon: 'architecture' as const },
  { to: '/comparison', label: 'Before / After', icon: 'comparison' as const },
];

const titles: Record<string, string> = {
  '/dashboard': 'Dashboard',
  '/architecture': 'Architecture',
  '/comparison': 'Before / After',
};

export function Layout() {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const location = useLocation();
  const admin = isAdmin(user);
  const shell = usePolling((signal) => api.get<MetricsResponse>('/system/metrics?history=1', signal), 5000, admin);

  if (!admin) {
    return <main className="min-h-screen bg-ink-950 px-4 py-8"><Outlet /></main>;
  }

  return (
    <div className="min-h-screen bg-ink-950 lg:flex">
      <aside className="flex shrink-0 flex-col border-b border-line bg-ink-900 lg:sticky lg:top-0 lg:h-screen lg:w-56 lg:border-b-0 lg:border-r">
        <div className="flex h-16 items-center gap-2.5 border-b border-line px-5">
          <span className="flex h-7 w-7 items-center justify-center rounded-sm bg-brand text-ink-950"><Icon name="database" size={17} /></span>
          <span className="text-sm font-semibold text-slate-100">MongoScale</span>
        </div>
        <nav aria-label="Main navigation" className="flex gap-1 overflow-x-auto p-3 lg:flex-1 lg:flex-col lg:overflow-visible">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) => classNames(
                'flex shrink-0 items-center gap-3 rounded-sm px-3 py-2.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand',
                isActive ? 'bg-ink-700 text-slate-100' : 'text-slate-400 hover:bg-ink-800 hover:text-slate-100',
              )}
            >
              <Icon name={item.icon} className="shrink-0" />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="hidden border-t border-line p-3 lg:block">
          <div className="truncate px-3 pb-2 text-sm font-medium text-slate-200">{user?.name}</div>
          <button type="button" className="btn-ghost w-full justify-start" onClick={logout}>
            <Icon name="logout" />
            Sign out
          </button>
        </div>
        <div className="flex items-center justify-between border-t border-line px-4 py-2 lg:hidden">
          <span className="truncate text-xs text-slate-400">{user?.name}</span>
          <button type="button" className="btn-ghost !px-2 !py-1.5" onClick={logout} aria-label="Sign out">
            <Icon name="logout" />
          </button>
        </div>
      </aside>
      <div className="min-w-0 flex-1">
        <header className="sticky top-0 z-20 flex min-h-16 flex-wrap items-center justify-between gap-3 border-b border-line bg-ink-950 px-5 py-3 lg:px-7">
          <h1 className="text-base font-semibold text-slate-100">{titles[location.pathname] ?? 'Dashboard'}</h1>
          <div className="flex flex-wrap items-center gap-2" aria-label="Current system configuration">
            {shell.data?.tuning ? <span className="rounded-sm border border-line px-2 py-1 text-xs text-slate-300">{shell.data.tuning.label}</span> : null}
            {shell.data?.topology ? <span className="rounded-sm border border-line px-2 py-1 text-xs text-slate-300">{shell.data.topology.kind}</span> : null}
          </div>
        </header>
        <main className="mx-auto w-full max-w-[1440px] px-4 py-6 lg:px-7 lg:py-7">
          {shell.error ? (
            <div role="status" aria-live="polite" className="mb-5 border-l-2 border-warn bg-ink-900 px-4 py-3 text-sm text-slate-200">
              API is unreachable. Live system data is temporarily unavailable.
            </div>
          ) : null}
          <Outlet />
        </main>
      </div>
    </div>
  );
}

export function RequireAuth() {
  const user = useAuthStore((state) => state.user);
  const location = useLocation();
  if (!isAdmin(user)) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return <Outlet />;
}
