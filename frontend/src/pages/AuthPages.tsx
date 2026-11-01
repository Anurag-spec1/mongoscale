import { useState, type FormEvent } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api, ApiError } from '../services/api';
import { useAuthStore } from '../store/auth';
import type { AuthResponse } from '../types';

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const setSession = useAuthStore((state) => state.setSession);
  const logout = useAuthStore((state) => state.logout);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const response = await api.post<AuthResponse>('/auth/login', { email, password });
      if (response.user.role !== 'admin') {
        logout();
        setError('This account does not have admin access.');
        return;
      }
      setSession(response);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from ?? '/dashboard', { replace: true });
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="mx-auto mt-10 max-w-md">
      <form onSubmit={submit} className="panel space-y-4 p-8">
        <div>
          <h1 className="text-2xl font-extrabold text-slate-100">Admin sign in</h1>
          <p className="mt-1 text-sm text-slate-500">Sign in with an administrator account to open the dashboard.</p>
        </div>
        <label className="block">
          <span className="label">Email</span>
          <input className="input" type="email" value={email} onChange={(event) => setEmail(event.target.value)} required autoComplete="email" />
        </label>
        <label className="block">
          <span className="label">Password</span>
          <input
            className="input"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            minLength={8}
            autoComplete="current-password"
          />
        </label>
        {error ? <div className="rounded-xl border border-danger/50 bg-danger/10 px-3 py-2 text-sm text-danger">{error}</div> : null}
        <button type="submit" className="btn-primary w-full" disabled={busy}>
          {busy ? 'Please wait…' : 'Sign in'}
        </button>
        <p className="rounded-xl border border-line bg-ink-900 px-3 py-2 text-xs text-slate-500">
          Use the administrator credentials configured for this deployment.
        </p>
      </form>
    </motion.div>
  );
}
