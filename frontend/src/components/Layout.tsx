import { Link, NavLink, Navigate, Outlet, useLocation } from 'react-router-dom';
import { isAdmin, useAuthStore } from '../store/auth';
import { classNames } from '../utils/format';

const linkClass = ({ isActive }: { isActive: boolean }) =>
  classNames('rounded-lg px-3 py-1.5 text-sm font-medium transition', isActive ? 'bg-brand/15 text-brand' : 'text-slate-400 hover:text-slate-100');

export function Layout() {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-line bg-ink-950/80 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center gap-6 px-4 py-3">
          <Link to="/" className="flex items-center gap-2 text-lg font-extrabold tracking-tight text-slate-100">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-ink-950">M</span>
            <span>
              Mongo<span className="text-brand">Scale</span>
            </span>
          </Link>
          <nav className="flex flex-1 flex-wrap items-center gap-1">
            {isAdmin(user) ? (
              <>
                <NavLink to="/dashboard" className={linkClass}>
                  Dashboard
                </NavLink>
                <NavLink to="/architecture" className={linkClass}>
                  Architecture
                </NavLink>
                <NavLink to="/comparison" className={linkClass}>
                  Before / After
                </NavLink>
              </>
            ) : null}
          </nav>
          {user ? (
            <div className="flex items-center gap-3">
              <span className="hidden text-sm text-slate-400 sm:block">{user.name}</span>
              <button type="button" className="btn-ghost !px-3 !py-1.5" onClick={logout}>
                Sign out
              </button>
            </div>
          ) : (
            <Link to="/login" className="btn-primary !px-3 !py-1.5">
              Sign in
            </Link>
          )}
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-8">
        <Outlet />
      </main>
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
