# postgres-operator
a simple kubernetes operator for postgreSQL for experiment

## Build
Need operator-sdk.

```
operator-sdk build fakemoon/postgres-operator:v0.0.1
docker push fakemoon/postgres-operator:v0.0.1
```
Current dockerhub release is fakemoon/postgres-operator:v0.0.6

## Deploy
```
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/operator.yaml
kubectl apply -f deploy/crds/fakemoon_v1alpha1_postgresdb_cr.yaml
```
