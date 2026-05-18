import { Navigate, Route, Routes } from 'react-router-dom';

import { Layout } from './components/Layout';
import { AlertsPage } from './pages/AlertsPage';
import { ApiStatusPage } from './pages/ApiStatusPage';
import { CrossExchangePage } from './pages/CrossExchangePage';
import { DashboardPage } from './pages/DashboardPage';
import { HealthPage } from './pages/HealthPage';
import { IBKRPage } from './pages/IBKRPage';
import { PricesPage } from './pages/PricesPage';
import { RelatedSignalsPage } from './pages/RelatedSignalsPage';
import { SettingsPage } from './pages/SettingsPage';
import { SpotFuturesPage } from './pages/SpotFuturesPage';
import { TriangularPage } from './pages/TriangularPage';

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<DashboardPage />} />
        <Route path="/prices" element={<PricesPage />} />
        <Route path="/triangular" element={<TriangularPage />} />
        <Route path="/cross-exchange" element={<CrossExchangePage />} />
        <Route path="/spot-futures" element={<SpotFuturesPage />} />
        <Route path="/signals" element={<RelatedSignalsPage />} />
        <Route path="/alerts" element={<AlertsPage />} />
        <Route path="/provider-health" element={<HealthPage />} />
        <Route path="/ibkr" element={<IBKRPage />} />
        <Route path="/status" element={<ApiStatusPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
}
