# Oracle Cloud Infrastructure (OCI) Knative Source

This source receives events from Oracle Cloud Event Service. It converts the event from cloudevent specification 0.1 to 1.0. To use it there need to be a [Oracle Cloud Notification](https://docs.oracle.com/en-us/iaas/Content/Notification/Tasks/managingtopicsandsubscriptions.htm) of type *HTTPS*.

This ContainerSource requires a Secret uses as Basic Authentication username and password:

*Basic Auth Secret Example*
```yaml
cat <<-EOF | kubectl apply -f -
---
apiVersion: v1
kind: Secret
metadata:
  name: oci-basic-auth
type: kubernetes.io/basic-auth
stringData:
  username: admin
  password: admin
EOF
```

This secret can be referenced in the ContainerSource description:

```yaml
cat <<-EOF | kubectl apply -f -
---
apiVersion: sources.knative.dev/v1
kind: ContainerSource
metadata:
  name: oci-source
spec:
  template:
    spec:
      containers:
        - image: localhost:5000/oci-source:v0.0.1
          name: oci-source
          env:
            - name: DEBUG
              value: "true"
            - name: BASICAUTH_USERNAME
              valueFrom:
                secretKeyRef:
                  name: oci-basic-auth
                  key: username
            - name: BASICAUTH_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: oci-basic-auth
                  key: password
  sink:
    ref:
      apiVersion: v1
      kind: Service
      name: event-display
EOF
```

To use this source it has to be exposed externally for Oracle to post data to this source. To do this there are two options; either create a service of type LoadBalancer or creating an Ingress in Kubernetes to an existing IngressController.

*Service Example*
```yaml
cat <<-EOF | kubectl apply -f -
---
apiVersion: v1
kind: Service
metadata:
  name: oci-service
spec:
  selector:
    sources.knative.dev/containerSource: oci-source
  ports:
    - port: 8000
EOF
```

If this source is used within a [Direktiv](https://github.com/direktiv/direktiv) instance it can be easily added to Kong's IngressController. The URL used in OCI would be *https://admin:admin@myserver.com/oci*

```yaml
cat <<-EOF | kubectl apply -f -
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: oci-ingress
  annotations:
    konghq.com/strip-path: "true"
spec:
  ingressClassName: kong
  rules:
  - host:
    http:
      paths:
        - path: /oci
          pathType: Prefix
          backend:
            service:
              name: oci-service
              port:
                number: 8000
EOF
```
