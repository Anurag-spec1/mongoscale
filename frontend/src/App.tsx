import { Navigate, Route, Routes } from 'react-router-dom';
import { Layout, RequireAuth } from './components/Layout';
import { isAdmin, useAuthStore } from './store/auth';
import ArchitecturePage from './pages/ArchitecturePage';
import ComparisonPage from './pages/ComparisonPage';
import DashboardPage from './pages/DashboardPage';
import { LoginPage } from './pages/AuthPages';

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomeRedirect />} />
        <Route path="login" element={<LoginPage />} />
        <Route element={<RequireAuth />}>
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="architecture" element={<ArchitecturePage />} />
          <Route path="comparison" element={<ComparisonPage />} />
        </Route>
        <Route path="*" element={<div className="py-24 text-center text-slate-500">Page not found</div>} />
      </Route>
    </Routes>
  );
}

function HomeRedirect() {
  const user = useAuthStore((state) => state.user);
  return <Navigate to={isAdmin(user) ? '/dashboard' : '/login'} replace />;
}
