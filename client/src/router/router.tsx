import LoginPage from "@/pages/auth/login";
import DashboardLayout from "@/pages/layout/DashbordLayout";
import RootLayout from "@/pages/layout/root";
import ServersIndex from "@/pages/servers";
import { createBrowserRouter, createRoutesFromElements, Route } from "react-router";
import ProtectedRoute from "./protectedRoute";


const router = createBrowserRouter(
  createRoutesFromElements(
    <Route path="/" element={<RootLayout></RootLayout>}>
      <Route path="login" element={<LoginPage></LoginPage>} ></Route>

      <Route element={<ProtectedRoute></ProtectedRoute>}>
        <Route path="" element={<DashboardLayout></DashboardLayout>} >
          <Route path="servers" element={<ServersIndex></ServersIndex>} ></Route>
        </Route>
      </Route>

    </Route>
  )
)

export default router