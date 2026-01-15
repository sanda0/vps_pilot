import * as React from "react"
import {
  FolderGit2,
  GalleryVerticalEnd,
  ReplaceAll,
  Server,
  Settings2,
} from "lucide-react"

import { NavMain } from "@/components/nav-main"
import { NavUser } from "@/components/nav-user"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenuButton,
  SidebarRail,
} from "@/components/ui/sidebar"
import { userAtom, User } from "@/atoms/user"
import { useAtom } from "jotai"

// This is sample data.
const data = {
  user: {
    name: "shadcn",
    email: "m@example.com",
    avatar: "/avatars/shadcn.jpg",
  },
  company:
  {
    name: "VPS Pilot",
    logo: GalleryVerticalEnd,
    plan: "Enterprise",
  }
  ,
  navMain: [
    {
      title: "Nodes",
      url: "/nodes",
      icon: Server,
    },
    {
      title: "Projects",
      url: "/projects",
      icon: FolderGit2
    },
    {
      title: "Cron Jobs",
      url: "#",
      icon: ReplaceAll
    },
    {
      title: "Settings",
      url: "#",
      icon: Settings2,
      items: [
        {
          title: "GitHub",
          url: "/settings/github",
        },
        {
          title: "Users",
          url: "#",
        }
      ],
    },
  ],
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {

  const [user] = useAtom<User | null>(userAtom)
  data.user.email = user?.email || data.user.email
  data.user.name = user?.username || data.user.name


  
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenuButton
          size="lg"
          className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
        >
          <div className="flex items-center justify-center rounded-lg aspect-square size-8 bg-sidebar-primary text-sidebar-primary-foreground">
            <data.company.logo className="size-4" />
          </div>
          <div className="grid flex-1 text-sm leading-tight text-left">
            <span className="font-semibold truncate">
              {data.company.name}
            </span>
            <span className="text-xs truncate">{data.company.plan}</span>
          </div>

        </SidebarMenuButton>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        {/* <NavProjects projects={data.projects} /> */}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={data.user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
