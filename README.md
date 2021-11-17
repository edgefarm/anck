# edgefarm-network-operator

## Generate Manifests

**Pre-Coditions:**

- operator-sdk installed

```bash
cd network-operator
make manifests
```

## Build and Push Operator Image

```bash
cd network-operator
make docker-build docker-push IMG="ci4rail/network-operator:latest"
```

## Deploy Operator

```bash
make deploy IMG="ci4rail/network-operator:latest"
```
