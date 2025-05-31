import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"
import MetricsTab from "./metrics-tab"
import AlertTab from "./alert-tab"




export function NodeViewTabs() {
  return (
    <Tabs defaultValue="metrics" className="w-full">
      <TabsList className="grid grid-cols-2 w-[fit-content]">
        <TabsTrigger value="metrics">Metrics</TabsTrigger>
        <TabsTrigger value="alerts">Alerts</TabsTrigger>
      </TabsList>
      <TabsContent value="metrics" className="p-2">
        <MetricsTab></MetricsTab>
      </TabsContent>
      <TabsContent value="alerts" className="p-2">
        <AlertTab></AlertTab>
      </TabsContent>
      
    </Tabs >
  )
}
