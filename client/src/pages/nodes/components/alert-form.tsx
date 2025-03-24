
import { Dialog, DialogHeader } from "@/components/ui/dialog";
import { DialogContent, DialogTitle } from "@/components/ui/dialog";
import { AlertFrom } from "@/forms/alert/alert";
import { Alert } from "@/models/alert";



interface AlertFromProps {
  open: boolean;
  isEdit: boolean;
  onOpenChange: (status: boolean) => void;
  onSaved?: () => void;
  alert?:Alert;
}

export default function AlertFromDialog(props: AlertFromProps) {



  return (<>
    <Dialog open={props.open} onOpenChange={props.onOpenChange} >
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Alert</DialogTitle>
        </DialogHeader>
        <div>
          <AlertFrom alert={props.alert} onFinished={()=>props.onOpenChange(false)}></AlertFrom>
        </div>

      </DialogContent>
    </Dialog>
  </>)
}