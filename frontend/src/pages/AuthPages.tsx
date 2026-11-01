import { useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api, ApiError } from '../services/api';
import { useAuthStore } from '../store/auth';
import type { AuthResponse } from '../types';

interface Props {
  mode: 'login' | 'register';
}

function AuthForm({ mode }: Props) {
  const navigate = useNavigate();
  const location = useLocation();
  const setSession = useAuthStore((state) => state.setSession);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const response =
        mode === 'login'
          ? await api.post<AuthResponse>('/auth/login', { email, password })
          : await api.post<AuthResponse>('/auth/register', { name, email, password });
      setSession(response);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from ?? (response.user.role === 'admin' ? '/dashboard' : '/'), { replace: true });
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
          <h1 className="text-2xl font-extrabold text-slate-100">{mode === 'login' ? 'Welcome back' : 'Create your account'}</h1>
          <p className="mt-1 text-sm text-slate-500">
            {mode === 'login' ? 'Sign in to place orders and open the performance dashboard.' : 'Customers can browse, review and order.'}
          </p>
        </div>
        {mode === 'register' ? (
          <label className="block">
            <span className="label">Name</span>
            <input className="input" value={name} onChange={(event) => setName(event.target.value)} required minLength={2} autoComplete="name" />
          </label>
        ) : null}
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
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          />
        </label>
        {error ? <div className="rounded-xl border border-danger/50 bg-danger/10 px-3 py-2 text-sm text-danger">{error}</div> : null}
        <button type="submit" className="btn-primary w-full" disabled={busy}>
          {busy ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Create account'}
        </button>
        <p className="text-center text-sm text-slate-500">
          {mode === 'login' ? (
            <>
              New here?{' '}
              <Link to="/register" className="text-brand hover:underline">
                Create an account
              </Link>
            </>
          ) : (
            <>
              Already registered?{' '}
              <Link to="/login" className="text-brand hover:underline">
                Sign in
              </Link>
            </>
          )}
        </p>
        {mode === 'login' ? (
          <p className="rounded-xl border border-line bg-ink-900 px-3 py-2 text-xs text-slate-500">
            Presenting? Sign in with the admin account from your <span className="font-mono text-slate-300">.env</span> (ADMIN_EMAIL / ADMIN_PASSWORD) to open the dashboard.
          </p>
        ) : null}
      </form>
    </motion.div>
  );
}

export function LoginPage() {
  return <AuthForm mode="login" />;
}

export function RegisterPage() {
  return <AuthForm mode="register" />;
}
