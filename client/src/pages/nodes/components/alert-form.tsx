import { Button } from "@/components/ui/button";
import { Dialog, DialogFooter, DialogHeader } from "@/components/ui/dialog";
import { DialogContent, DialogTitle } from "@/components/ui/dialog";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertSchema } from "@/schema/node";
import { FormField, FormItem, FormLabel, FormControl, FormMessage, Form } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface AlertFromProps {
  open: boolean;
  isEdit: boolean;
  onOpenChange: (status: boolean) => void;
  onSaved?: () => void;
}

export default function AlertFrom(props: AlertFromProps) {

  const form = useForm({
    resolver: zodResolver(AlertSchema),
    defaultValues: {
      metric: "CPU",
      threshold: 70,
      duration: 5,
      discord: "",
      enabled: true,
    },
  });

  const onSubmit = (data: any) => {
    console.log("Alert Config:", data);

  };

  return (<>
    <Dialog open={props.open} onOpenChange={props.onOpenChange} >
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{props.isEdit ? "Edit" : "Create"} Alert</DialogTitle>
        </DialogHeader>
        <div>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="max-w-md mx-auto space-y-4 ">
              {/* Metric Selection */}
              <FormField control={form.control} name="metric" render={({ field }) => (
                <FormItem>
                  <FormLabel>Metric</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select metric" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="CPU">CPU</SelectItem>
                      <SelectItem value="Memory">Memory</SelectItem>
                      {/* <SelectItem value="Network">Network</SelectItem> */}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )} />

              {/* Threshold */}
              <FormField control={form.control} name="threshold" render={({ field }) => (
                <FormItem>
                  <FormLabel>Threshold (%)</FormLabel>
                  <FormControl>
                    <Input type="number" placeholder="Enter threshold" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )} />

              {/* Duration */}
              <FormField control={form.control} name="duration" render={({ field }) => (
                <FormItem>
                  <FormLabel>Duration (Minutes)</FormLabel>
                  <FormControl>
                    <Input type="number" placeholder="Enter duration" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )} />



              {/* Webhook */}
              <FormField control={form.control} name="discord" render={({ field }) => (
                <FormItem>
                  <FormLabel>Discord Webhook URL</FormLabel>
                  <FormControl>
                    <Input type="url" placeholder="Enter Discord webhook URL (optional)" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )} />

              {/* Enable Alert */}
              <FormField control={form.control} name="enabled" render={({ field }) => (
                <FormItem className="flex items-center justify-between">
                  <FormLabel>Enable Alert</FormLabel>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )} />

              {/* Submit Button */}
              {/* <Button type="submit" className="w-full">Create Alert</Button>
               */}
              <DialogFooter>
                <Button type="submit">Save</Button>
              </DialogFooter>
            </form>
          </Form>
        </div>

      </DialogContent>
    </Dialog>
  </>)
}