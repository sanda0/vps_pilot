import LoginPage from "@/pages/auth/login";
import DashboardLayout from "@/pages/layout/DashbordLayout";
import RootLayout from "@/pages/layout/root";
import { createBrowserRouter, createRoutesFromElements, Route } from "react-router";
import ProtectedRoute from "./protectedRoute";
import NodesIndex from "@/pages/nodes";
import NodeView from "@/pages/nodes/view";
import ProjectsListPage from "@/pages/projects";
import CreateProjectPage from "@/pages/projects/create";
import ProjectDetailsPage from "@/pages/projects/[id]";
import EditProjectPage from "@/pages/projects/[id]/edit";
import GitHubSettings from "@/pages/settings/github";


const router = createBrowserRouter(
  createRoutesFromElements(
    <Route path="/" element={<RootLayout></RootLayout>}>
      <Route path="login" element={<LoginPage></LoginPage>} ></Route>

      <Route element={<ProtectedRoute></ProtectedRoute>}>
        <Route path="" element={<DashboardLayout></DashboardLayout>} >
          <Route path="nodes" element={<NodesIndex></NodesIndex>} ></Route>
          <Route path="nodes/:id" element={<NodeView></NodeView>} ></Route>
          
          <Route path="projects" element={<ProjectsListPage></ProjectsListPage>} ></Route>
          <Route path="projects/create" element={<CreateProjectPage></CreateProjectPage>} ></Route>
          <Route path="projects/:id" element={<ProjectDetailsPage></ProjectDetailsPage>} ></Route>
          <Route path="projects/:id/edit" element={<EditProjectPage></EditProjectPage>} ></Route>
          
          <Route path="settings/github" element={<GitHubSettings></GitHubSettings>} ></Route>
        </Route>
      </Route>

    </Route>
  )
)

export default router