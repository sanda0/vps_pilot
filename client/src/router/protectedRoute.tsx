import { Navigate, Outlet } from "react-router"

export default function ProtectedRoute(){
  const user = true
  return user ? <Outlet></Outlet> : <Navigate to="/login"></Navigate>
}