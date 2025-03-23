import { Button } from "@/components/ui/button";
import { useEffect, useState } from "react";
import AlertFrom from "./alert-form";
import { Alert } from "@/models/alert";
import { useParams } from "react-router";
import api from "@/lib/api";
import AlertCard from "./alert-card";


export default function AlertTab() {

  const [openFormDialog, setFromDilaog] = useState(false)
  const { id } = useParams<{ id: string }>();
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [currentAlert, setCurrentAlert] = useState<Alert >()

  const loadAlerts = () => {
    api.get(`/alerts?node_id=${id}&limit=100&offset=0`).then((res) => {
      setAlerts(res.data.data)
    })

  }

  useEffect(() => {
    loadAlerts()
  }, [])

  useEffect(() => {
    loadAlerts()
  }, [openFormDialog])

  const onAlertDelete = (id: number) => {
    api.delete(`/alerts/${id}`).then(() => {
      loadAlerts()
    })
  }

  return <>
    <div>
      <AlertFrom open={openFormDialog} alert={currentAlert} isEdit={false} onOpenChange={(e) => { setFromDilaog(e) }}  ></AlertFrom>
      <div className="flex justify-between">
        <div className="text-2xl font-semibold">Alerts</div>
        <div className="flex items-center ">
          <Button onClick={() => setFromDilaog(true)}>Create Alert</Button>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 mt-4 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
        {alerts.map((alert) => {
          return <AlertCard
            key={alert.id}
            id={alert.id}
            matric={alert.metric}
            value={alert.threshold.Float64}
            isEnable={alert.is_active.Bool}
            net_recv={alert.net_rece_threshold.Float64}
            net_sent={alert.net_send_threshold.Float64}
            email={alert.email.String}
            discord={alert.discord_webhook.String}
            slack={alert.slack_webhook.String}
            onEditClick={() => { }}
            onDeleteClick={onAlertDelete}
          ></AlertCard>
        })}
      </div>
    </div>
  </>
}