"use client"
import * as z from "zod"
import {
  formSchema
} from "@/forms/alert/schema"

import {
  zodResolver
} from "@hookform/resolvers/zod"
import {
  Button
} from "@/components/ui/button"
import {
  useForm
} from "react-hook-form"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select"
import {
  Input
} from "@/components/ui/input"
import {
  Switch
} from "@/components/ui/switch"
import { useState } from "react"
import api from "@/lib/api"
import { useParams } from "react-router"
import { Alert } from "@/models/alert"

interface AlertFromProps {
  onFinished: () => void;
  alert?: Alert;
}

export function AlertFrom(props: AlertFromProps) {

  const { id } = useParams<{ id: string }>();

  const [isPending, setIsPending] = useState(false)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: props.alert?.id,
      node_id: Number(id),
      metric: props.alert?.metric,
      threshold: props.alert?.threshold.Float64,
      net_rece_threshold: props.alert?.net_rece_threshold.Float64,
      net_send_threshold: props.alert?.net_send_threshold.Float64,
      duration: props.alert?.duration,
      email: props.alert?.email.String,
      discord: props.alert?.discord_webhook.String,
      slack: props.alert?.slack_webhook.String,
      enabled: props.alert?.is_active.Bool

    },
  })

  const onSubmit = () => {

    setIsPending(true)
    console.log("Submit")
    console.log(form.getValues())
    if (props.alert?.id) {
      api.put(`/alerts`, form.getValues()).then((res) => {
        console.log(res)
        setIsPending(false)
        props.onFinished()
      }
      )
    } else {
      api.post('/alerts', form.getValues()).then((res) => {
        console.log(res)
        setIsPending(false)
        props.onFinished()
      })
    }
  }

  return (
    <div>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col w-full max-w-3xl gap-2 p-2 mx-auto rounded-md md:p-5">


          <FormField
            control={form.control}
            name="metric"
            render={({ field }) => {
              const options = [
                { value: 'cpu', label: 'CPU' },
                { value: 'mem', label: 'Memory' },
                { value: 'net', label: 'Network' },
              ]
              return (
                <FormItem>
                  <FormLabel>Select Matric</FormLabel> *
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select Matric" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {options.map(({ label, value }) => (
                        <SelectItem key={value} value={value}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <FormMessage />
                </FormItem>
              )
            }}
          />
          {(form.watch("metric") === "cpu" || form.watch("metric") === "mem") && <FormField
            control={form.control}
            name="threshold"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Threshold(%)</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Threshold"
                    type={"number"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(+val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />}

          {form.watch("metric") === "net" && <><FormField
            control={form.control}
            name="net_rece_threshold"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Network Receive Threshold</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Network Receive Threshold"
                    type={"number"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(+val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />
            <FormField
              control={form.control}
              name="net_send_threshold"
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormLabel>Network Send Threshold</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Network Send Threshold"
                      type={"number"}
                      value={field.value}
                      onChange={(e) => {
                        const val = e.target.value;
                        field.onChange(+val);
                      }}
                    />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )
              }
            /> </>}
          <FormField
            control={form.control}
            name="duration"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Duration</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Duration"
                    type={"number"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(+val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Email</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Email"
                    type={"email"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />
          <FormField
            control={form.control}
            name="discord"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Discord Webhook</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Discord Webhook"
                    type={"url"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />
          <FormField
            control={form.control}
            name="slack"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Slack Webhook</FormLabel>
                <FormControl>
                  <Input
                    placeholder="Slack Webhook"
                    type={"url"}
                    value={field.value}
                    onChange={(e) => {
                      const val = e.target.value;
                      field.onChange(val);
                    }}
                  />
                </FormControl>

                <FormMessage />
              </FormItem>
            )
            }
          />

          <FormField
            control={form.control}
            name="enabled"
            render={({ field }) => (
              <FormItem className="flex flex-col justify-center w-full p-3 border rounded">
                <div className="flex items-center justify-between h-full">
                  <FormLabel>Enabled</FormLabel>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </div>

              </FormItem>
            )}
          />
          <div className="flex items-center justify-end w-full pt-3">
            <Button className="rounded-lg" size="sm">
              {isPending ? 'Submitting...' : 'Submit'}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}