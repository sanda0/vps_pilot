import LoginPage from "@/pages/auth/login";
import DashboardLayout from "@/pages/layout/DashbordLayout";
import RootLayout from "@/pages/layout/root";
import { createBrowserRouter, createRoutesFromElements, Route } from "react-router";
import ProtectedRoute from "./protectedRoute";
import NodesIndex from "@/pages/nodes";
import NodeView from "@/pages/nodes/view";


const router = createBrowserRouter(
  createRoutesFromElements(
    <Route path="/" element={<RootLayout></RootLayout>}>
      <Route path="login" element={<LoginPage></LoginPage>} ></Route>

      <Route element={<ProtectedRoute></ProtectedRoute>}>
        <Route path="" element={<DashboardLayout></DashboardLayout>} >
          <Route path="nodes" element={<NodesIndex></NodesIndex>} ></Route>
          <Route path="nodes/:id" element={<NodeView></NodeView>} ></Route>
        </Route>
      </Route>

    </Route>
  )
)

export default router