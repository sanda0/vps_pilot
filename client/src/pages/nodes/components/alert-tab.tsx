import { Button } from "@/components/ui/button";
import { useState } from "react";
import AlertFrom from "./alert-form";


export default function AlertTab() {

  const [openFormDialog,setFromDilaog] = useState(false)

  return <>
    <div>
      <AlertFrom open={openFormDialog} isEdit={false} onOpenChange={(e)=>{setFromDilaog(e)}}  ></AlertFrom>
      <div className="flex justify-between">
        <div className="text-2xl font-semibold">Alerts</div>
        <div className="flex items-center ">
          <Button onClick={()=>setFromDilaog(true)}>Create Alert</Button>
        </div>
      </div>
    </div>
  </>
}