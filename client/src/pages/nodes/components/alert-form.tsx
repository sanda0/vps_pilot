import { Button } from "@/components/ui/button";
import { Dialog, DialogFooter, DialogHeader } from "@/components/ui/dialog";
import { DialogContent, DialogTitle } from "@/components/ui/dialog";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
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
          <AlertFrom></AlertFrom>
        </div>

      </DialogContent>
    </Dialog>
  </>)
}