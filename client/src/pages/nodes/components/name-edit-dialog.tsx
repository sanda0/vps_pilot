import { Button } from "@/components/ui/button";
import { DialogHeader, DialogFooter, Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import api from "@/lib/api";
import { Label } from "@radix-ui/react-label";
import { useEffect, useState } from "react";


interface NameEditDialogProps {
  open: boolean;
  onOpenChange: (status: boolean) => void;
  name?: string;
  id?: number;
  onSaved?: () => void;
}

export function NameEditDialog(props: NameEditDialogProps) {
  const [name, setName] = useState(props.name)
  // setName(props.name)
  useEffect(() => {
    setName(props.name)
  }, [props.name])

  const handleSave = () => {
    api.put('/nodes/change-name', {
      id: props.id,
      name: name
    }).then((res) => {
      if (res.status === 200) {
        props.onOpenChange(false)
        props.onSaved?.()
      }
    }).catch((err) => {
      console.error(err)
    })
  }

  return <>
    <Dialog open={props.open} onOpenChange={(v) => props.onOpenChange(v)} >
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Edit Node Name</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid items-center grid-cols-4 gap-4">
            <Label htmlFor="name" className="col-span-4">Name</Label>
            <Input id="name" className="col-span-4" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
        </div>
        <DialogFooter>
          <Button type="submit" onClick={handleSave}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </>

}