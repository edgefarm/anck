# edgefarm-network-operator

## Generate Manifests

**Pre-Coditions:**

- operator-sdk installed

```bash
make manifests
```

## Build and Push Operator Image

```bash
make docker-build docker-push IMG="ci4rail/edgefarm-network-operator:latest"
```

## Deploy Operator

```bash
make deploy IMG="ci4rail/edgefarm-network-operator:latest"
```
