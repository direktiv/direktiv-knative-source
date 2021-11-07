


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

Service
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

Ingress
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
