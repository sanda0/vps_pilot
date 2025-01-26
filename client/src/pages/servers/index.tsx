import { Button } from "@/components/ui/button";
import { BadgePlus } from "lucide-react";


export default function ServersIndex() {
  return (
    < >
      <div className="w-full h-16 max-w-5xl mx-auto rounded-xl ">
        <div className="flex items-center justify-between h-full px-4">
          <h1 className="text-2xl font-semibold">Servers</h1>
          <Button>
            <BadgePlus></BadgePlus>
            Add Server
          </Button>
        </div>
      </div>
      <div className="w-full h-full max-w-5xl mx-auto rounded-xl bg-muted/50" >

      </div>

    </>
  )
}