# restcomm-zabbix-module

Shared object module for zabbix

It collects restcomm metrics from all restcomm nodes(tasks) in mesos cluster. 

### Discovery
You should use LLD with `restcomm.discovery` key

Discovery result contains the followings keys

1. `{#APP_NAME}` - cluster app name. `restcomm` by default
2. `{#INSTANCE_ID}` - restocmm node Instance id

### Available metrics:
metrics template is `restcomm.metrics[{#INSTANCE_ID},<Metric Key>]`

Metric Keys:

1. LiveCalls
2. TotalCallsSinceUptime
3. FailedCalls
4. CompletedCalls
5. LiveOutgoingCalls
6. LiveIncomingCalls
