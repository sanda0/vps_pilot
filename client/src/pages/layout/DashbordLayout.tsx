import { AppSidebar } from "@/components/app-sidebar"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
// import {
//   Breadcrumb,
//   BreadcrumbItem,
//   BreadcrumbLink,
//   BreadcrumbList,
//   BreadcrumbPage,
//   BreadcrumbSeparator,
// } from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { useState } from "react"
import { Outlet } from "react-router"

export default function DashboardLayout() {

  
  const {theme,setTheme} = useTheme()
  const [themeEmoji, setThemeEmoji] = useState(theme === "light" ? "🌙" : "🌞")

  const toggleTheme = () => {
    if (theme === "light") {
      setThemeEmoji("🌞")
      setTheme("dark")
    } else {
      setThemeEmoji("🌙")
      setTheme("light")
    }
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12">
          <div className="flex items-center justify-between w-full gap-2 px-4">
            <div className="flex items-center gap-2">
              <SidebarTrigger className="-ml-1" />
              <Separator orientation="vertical" className="h-4 mr-2" />
            </div>
            {/* <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="#">
                    Building Your Application
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Data Fetching</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb> */}
            <div>
            <Button size={"icon"} className="text-xl" variant={"outline"} onClick={toggleTheme} >{themeEmoji}</Button>
            </div>
          </div>
        </header>
        <div className="flex flex-col flex-1 gap-4 px-4 py-10">
          <Outlet></Outlet>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
