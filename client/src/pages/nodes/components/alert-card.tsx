import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Edit, Trash2 } from "lucide-react";
import { SocialIcon } from "react-social-icons";


interface AlertCardProps {
  id: number;
  isEnable: boolean;
  matric: string;
  value: number;
  net_sent: number;
  net_recv: number;
  email?: string;
  discord?: string;
  slack?: string;
  onEditClick?: (id: number) => void;
  onDeleteClick?: (id: number) => void;
}

export default function AlertCard(props: AlertCardProps) {
  return <>
    <Card className="w-full">
      <CardContent>
        <div className="flex justify-between pt-4">
          <div>
            {props.isEnable ? <Badge variant={"default"}>Active</Badge> : <Badge variant={"destructive"}>Inactive</Badge>}

          </div>
          <div className="flex gap-2">
            <Button size={"icon"} variant={"secondary"} onClick={() => { props.onEditClick && props.onEditClick(props.id) }}><Edit></Edit></Button>

            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button size={"icon"} variant={"destructive"}><Trash2></Trash2></Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This action cannot be undone. This will permanently delete alert.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction onClick={() => { props.onDeleteClick && props.onDeleteClick(props.id) }} >Continue</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
        <div className="flex justify-between">
          <div>
            <div className="text-lg font-semibold">
              {props.matric == "cpu" ? "CPU" : null}
              {props.matric == "mem" ? "Memory" : null}
              {props.matric == "net" ? "Network" : null}
            </div>
            <div className="text-sm text-gray-500">{props.matric == "net" ? "Sent: " + props.net_sent + " B/s | Recv: " + props.net_recv + " B/s" : props.value + "%"}</div>
          </div>
          <div className="flex items-center gap-2">
            {props.email == "" ? null : <SocialIcon network="email" style={{ height: 32, width: 32 }}></SocialIcon>}
            {props.discord == "" ? null : <SocialIcon network="discord" style={{ height: 32, width: 32 }}></SocialIcon>}
            {props.slack == "" ? null : <SocialIcon network="slack" style={{ height: 32, width: 32 }}></SocialIcon>}

          </div>
        </div>

      </CardContent>
    </Card>
  </>
}