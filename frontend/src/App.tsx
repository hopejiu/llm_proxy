import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AppProvider } from "./context/AppContext";
import { ToastProvider } from "./components/Toast";
import Layout from "./components/Layout";
import ErrorBoundary from "./components/ErrorBoundary";
import ProvidersPage from "./pages/ProvidersPage";
import StatsPage from "./pages/StatsPage";
import LogsPage from "./pages/LogsPage";
import RealtimePage from "./pages/RealtimePage";
import SettingsPage from "./pages/SettingsPage";

export default function App() {
  return (
    <BrowserRouter>
      <AppProvider>
        <ToastProvider>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Navigate to="/providers" replace />} />
              <Route path="providers" element={<ErrorBoundary><ProvidersPage /></ErrorBoundary>} />
              <Route path="stats" element={<ErrorBoundary><StatsPage /></ErrorBoundary>} />
              <Route path="logs" element={<ErrorBoundary><LogsPage /></ErrorBoundary>} />
              <Route path="realtime" element={<ErrorBoundary><RealtimePage /></ErrorBoundary>} />
              <Route path="settings" element={<ErrorBoundary><SettingsPage /></ErrorBoundary>} />
            </Route>
          </Routes>
        </ToastProvider>
      </AppProvider>
    </BrowserRouter>
  );
}
