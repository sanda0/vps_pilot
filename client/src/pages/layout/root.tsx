import { Outlet } from "react-router";
import { Toaster } from "@/components/ui/toaster";


export default function RootLayout() {

  return (
    <>
      <Outlet />
      <Toaster />
    </>
  )

}