import api from "@/lib/api"
import { useState, useEffect } from "react"
import { Navigate, Outlet } from "react-router"

export default function ProtectedRoute() {
  const [isLogged, setIsLogged] = useState<boolean | null>(null); // 🔹 null for "loading" state

  useEffect(() => {
    api.get("/profile")
      .then((res) => {
        if (res.status === 200) {
          setIsLogged(true);
        }
      })
      .catch(() => {
        setIsLogged(false); // 🔹 Set to false if request fails
      });
  }, []);

  // 🔹 Show loading while checking authentication
  if (isLogged === null) {
    return <div>Loading...</div>; // You can replace this with a spinner
  }

  return isLogged ? <Outlet /> : <Navigate to="/login" />;
}
