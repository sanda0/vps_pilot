import { userAtom,User } from "@/atoms/user";
import api from "@/lib/api"
import { useAtom } from "jotai";
import { useState, useEffect } from "react"
import { Navigate, Outlet } from "react-router"

export default function ProtectedRoute() {
  const [isLogged, setIsLogged] = useState<boolean | null>(null); // 🔹 null for "loading" state
  const [_, setUserAtom] = useAtom<User | null>(userAtom)

  useEffect(() => {
    api.get("/profile")
      .then((res) => {
        if (res.status === 200) {
          setIsLogged(true);
          setUserAtom({
            id: res.data.data.id,
            email: res.data.data.email,
            token: res.data.data.token,
            username: res.data.data.username
          });
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
