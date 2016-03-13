set -ex

mkdir -p output
go build -buildmode=c-shared -o output/restcomm-zabbix-module.so

echo "Copy so to juju"
juju scp output/restcomm-zabbix-module.so mesos-master/1:
