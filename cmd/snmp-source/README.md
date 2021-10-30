# SNMP UDP Knative Source

This source receives SNMP traps and converts them into cloudevents to be consumed by Knative. It starts the trap listener on port 8000. If the traps are coming from outside this ContainerSource needs to be exposed via a service.

Knative Eventing uses the label *sources.knative.dev/containerSource* for the ContainerSource pods and this can be used to create a Nodeport to receive the traps.


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

To test the ser

sudo apt install snmp snmp-mibs-downloader


sudo snmptrap -v 2c -c "Tas" 10.0.0.104:30007 0 1.3.6.1.4.1.2.3 1.3.6.1.6.1.4.1.2.3.1.1.1.1.1 s "This is a Test"

sudo snmptrap -v 2c -c public 10.0.0.104:30007 SNMPv2-MIB::coldStart 1.3.6.1.6.3.1.1.5.1


/**
 * TrapType defines the type of SNMPv2/SNMPv3 trap,
 * this is defined in the SNMPv2-MIB as snmpTrapOID.0
 * (.1.3.6.1.6.3.1.1.4.1.0) with an OID value of one
 *  of the following
 */
public static final String SNMP_TRAP_OID = "1.3.6.1.6.3.1.1.4.1.0";

/** coldStart OID */
public static final String COLDSTART_OID = "1.3.6.1.6.3.1.1.5.1";

/** warmStart OID */
public static final String WARMSTART_OID = "1.3.6.1.6.3.1.1.5.2";

/** linkDown OID */
public static final String LINKDOWN_OID = "1.3.6.1.6.3.1.1.5.3";

/** linkUp OID */
public static final String LINKUP_OID = "1.3.6.1.6.3.1.1.5.4"
