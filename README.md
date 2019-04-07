# Prerequisites

Create a Azure SP that has Owner level permissions to the desired WAF instance.
Then create a secret on the cluster that is going to run the director.

```
kubectl create secret generic -n kube-system ace-waf-director \
    --from-literal="client_id=$AZURE_CLIENT_ID" \
    --from-literal="client_secret=$AZURE_CLIENT_SECRET" \
    --from-literal="tenant_id=$AZURE_TENANT_ID" \
    --from-literal="subscription_id=$AZURE_SUBSCRIPTION_ID"
```

Use the helm chart to install it into k8s.
