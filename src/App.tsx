import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { I18nextProvider } from "react-i18next";
import i18n from "@/i18n";
import Index from "./pages/Index";
import Login from "./pages/Login";
import Register from "./pages/Register";
import NotFound from "./pages/NotFound";
import DashboardLayout from "./components/DashboardLayout";
import Overview from "./pages/dashboard/Overview";
import Assets from "./pages/dashboard/Assets";
import Security from "./pages/dashboard/Security";
import Lending from "./pages/dashboard/Lending";
import Addresses from "./pages/dashboard/Addresses";
import Withdraw from "./pages/dashboard/Withdraw";

const queryClient = new QueryClient();

const App = () => (
  <I18nextProvider i18n={i18n}>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Toaster />
        <Sonner />
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Index />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            
            <Route path="/dashboard" element={<DashboardLayout />}>
              <Route index element={<Overview />} />
              <Route path="assets" element={<Assets />} />
              <Route path="lending" element={<Lending />} />
              <Route path="addresses" element={<Addresses />} />
              <Route path="withdraw" element={<Withdraw />} />
              <Route path="investments" element={<div>Investments Page (Coming Soon)</div>} />
              <Route path="security" element={<Security />} />
              <Route path="statements" element={<div>Statements Page (Coming Soon)</div>} />
            </Route>

            {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
            <Route path="*" element={<NotFound />} />
          </Routes>
        </BrowserRouter>
      </TooltipProvider>
    </QueryClientProvider>
  </I18nextProvider>
);

export default App;
