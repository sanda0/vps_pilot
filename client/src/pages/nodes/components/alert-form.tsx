
import { Dialog, DialogHeader } from "@/components/ui/dialog";
import { DialogContent, DialogTitle } from "@/components/ui/dialog";
import { AlertFrom } from "@/forms/alert/alert";



interface AlertFromProps {
  open: boolean;
  isEdit: boolean;
  onOpenChange: (status: boolean) => void;
  onSaved?: () => void;
}

export default function AlertFromDialog(props: AlertFromProps) {



  return (<>
    <Dialog open={props.open} onOpenChange={props.onOpenChange} >
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Alert</DialogTitle>
        </DialogHeader>
        <div>
          <AlertFrom onFinished={()=>props.onOpenChange(false)}></AlertFrom>
        </div>

      </DialogContent>
    </Dialog>
  </>)
}