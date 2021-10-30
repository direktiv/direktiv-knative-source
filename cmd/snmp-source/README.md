# SNMP UDP Knative Source

This source receives SNMP traps and converts them into cloudevents to be consumed by Knative. It starts the trap listener on port 8000. If the traps are coming from outside this ContainerSource needs to be exposed via a service.

Knative Eventing uses the label *sources.knative.dev/containerSource* for the ContainerSource pods and it can be used to create a Nodeport to receive the traps from external trap sources.


*Nodeport Example, Target port 30007*
```
cat <<-EOF | kubectl apply -f -
---
apiVersion: v1
kind: Service
metadata:
  name: snmp-service
spec:
  type: NodePort
  selector:
    sources.knative.dev/containerSource: snmp-source
  ports:
    - port: 8000
      protocol: UDP
      targetPort: 8000
      nodePort: 30007
EOF
```

## Test

The source can be tested with *snmptrap* on Linux.

```console
sudo apt install snmp snmp-mibs-downloader
```

Sending trap examples:

```console
snmptrap -v 2c -c "Tas" 127.0.0.1:30007 0 1.3.6.1.4.1.2.3 1.3.6.1.6.1.4.1.2.3.1.1.1.1.1 s "This is a Test"

snmptrap -v 2c -c public 127.0.0.1:30007 SNMPv2-MIB::coldStart 1.3.6.1.6.3.1.1.5.1
```
