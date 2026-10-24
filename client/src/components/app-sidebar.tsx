import * as React from "react"
import {
  FolderGit2,
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
          {/* Same wordmark as the website: a ">_" prompt, then the name. */}
          <div className="flex items-center justify-center font-mono font-bold aspect-square size-8 text-[hsl(var(--viz-1))]" aria-hidden="true">
            &gt;<span className="wordmark-cursor">_</span>
          </div>
          <span className="flex-1 font-semibold tracking-tight truncate">VPS Pilot</span>

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
