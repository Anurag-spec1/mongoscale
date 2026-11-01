import { Route, Routes } from 'react-router-dom';
import { Layout, RequireAuth } from './components/Layout';
import ArchitecturePage from './pages/ArchitecturePage';
import ComparisonPage from './pages/ComparisonPage';
import DashboardPage from './pages/DashboardPage';
import { LoginPage } from './pages/AuthPages';

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="login" element={<LoginPage />} />
        <Route element={<RequireAuth admin />}>
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="architecture" element={<ArchitecturePage />} />
          <Route path="comparison" element={<ComparisonPage />} />
        </Route>
        <Route path="*" element={<div className="py-24 text-center text-slate-500">Page not found</div>} />
      </Route>
    </Routes>
  );
}
