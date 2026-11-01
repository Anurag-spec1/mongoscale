import { useState, type FormEvent } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api, ApiError } from '../services/api';
import { useAuthStore } from '../store/auth';
import type { AuthResponse } from '../types';
import { Icon } from '../components/icons';

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
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.18 }} className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-sm items-center">
      <form onSubmit={submit} className="w-full space-y-5 border border-line bg-ink-900 p-6 sm:p-8">
        <div className="mb-7">
          <div className="mb-5 flex items-center gap-2 text-sm font-semibold text-slate-100">
            <span className="flex h-7 w-7 items-center justify-center rounded-sm bg-brand text-ink-950"><Icon name="database" size={17} /></span>
            MongoScale
          </div>
          <h1 className="text-2xl font-semibold text-slate-100">Admin sign in</h1>
          <p className="mt-2 text-sm text-slate-400">Sign in with an administrator account.</p>
        </div>
        <label className="block">
          <span className="label">Email address</span>
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
        {error ? <div role="alert" className="border-l-2 border-danger bg-status-danger/10 px-3 py-2 text-sm text-danger">{error}</div> : null}
        <button type="submit" className="btn-primary w-full" disabled={busy}>
          {busy ? 'Please wait…' : 'Sign in'}
        </button>
      </form>
    </motion.div>
  );
}
