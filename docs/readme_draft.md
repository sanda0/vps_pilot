# vps pilot

## Monitoring

agents install on eact nodes then nodes(servers) send matrix to central dashbord (cpu,memory,network matrix) shown them in charts with 5m,15m,1h,1d,2d,7d time slots

## alerting

give option to config alert to user get notification from email,and discord. how it work 1st user config alert with matrix threshhold point

```
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
```

slack and mail not implement yet

## projects

i need to creat option manage projects on eact node.
nodes can have multiple project. developers need to add `config.vpspilot.json` file to projects then agent shearch that config files on disk and send that data to central server when developer chage that config file(s) updates data sent to central server

`config.vpspilot.json` file look like this

```json

{
"name":"meta ads dashboard",
"tech":["laravel","react","mysql"], // optional
"logs":[],//path to project logs folders
"commands":[
  {
    "name":"node build",
    "command":"npm run build"
  },
    {
    "name":"php build",
    "command":"composer install"
  },
],//this commands can run from central server (show them in like dropdown and run button)
"backups":{
    "env_file": ".env",
    "zip_file_name": "project_backup",
    "database": {
        "connection": "DB_CONNECTION",
        "host": "DB_HOST",
        "port": "DB_PORT",
        "username": "DB_USERNAME",
        "password": "DB_PASSWORD",
        "database_name": "DB_DATABASE"
    },
    "dir": [
        "storage/app",
        "database/companies"
    ]
  }// using this can backup giving dirs and data base backup
}

```

NOTE: this project part is not implement yet i give my plan add this like todo in read me

## cron jobs

this is option to create cron jobs in nodes from central server not implement yet add this todo also


