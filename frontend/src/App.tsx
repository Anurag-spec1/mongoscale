import { Route, Routes } from 'react-router-dom';
import { Layout, RequireAuth } from './components/Layout';
import ArchitecturePage from './pages/ArchitecturePage';
import CartPage from './pages/CartPage';
import CatalogPage from './pages/CatalogPage';
import ComparisonPage from './pages/ComparisonPage';
import DashboardPage from './pages/DashboardPage';
import { LoginPage, RegisterPage } from './pages/AuthPages';
import OrdersPage from './pages/OrdersPage';
import ProductPage from './pages/ProductPage';

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<CatalogPage />} />
        <Route path="products/:id" element={<ProductPage />} />
        <Route path="cart" element={<CartPage />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route path="orders" element={<OrdersPage />} />
        </Route>
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
