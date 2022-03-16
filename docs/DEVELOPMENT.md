# How to develop and debug `anck`

Using devspace, vscode and dlv you can develop and debug anck.

## Settings in vscode

Use this section in your launch.json file to debug the anck-controller.

### launch.json for anch-controller

You need to open the project `anck` in vscode and use the following settings:

```json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "anck-controller localhost:23451",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "port": 23451,
            "host": "localhost",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}/",
                    "to": "/app/",
                },
            ],
            "showLog": true,
            "stopOnEntry": false,
            "trace": "verbose", // use for debugging problems with delve (breakpoints not working, etc.)
        }
    ]
}
```

### launch.json for anch-credentials

You need to open the project `anck-credentials` in vscode and use the following settings:

```json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "anck-credentials localhost:23450",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "port": 23450,
            "host": "localhost",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}/",
                    "to": "/app/",
                },
            ],
            "showLog": true,
            "stopOnEntry": false,
            "trace": "verbose", // use for debugging problems with delve (breakpoints not working, etc.)
        }
    ]
}
```

## How to develop anck

This will create a devspace dev environment where the pods for the anck-controller and anck-credentials are swapped by debugging containers running an golang image.

```bash
devspace dev
```

## Updating CRDs

If you made any changes to the CRDs you can install them using the following command:

```bash
make install
```

## Running while developing

Start running for e.g. anck-credentials and start the program.

```bash
$ kubectl ns anck
Active namespace is "anck".
$ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
anck-controller-manager-666cb6bf45-gztcs-2rr9m   2/2     Running   0          11h
anck-credentials-774f6cc648-pd5vz-zm5bm          1/1     Running   0          11h

# Start one shell with anck-credentials
$ kubectl exec -it anck-credentials-774f6cc648-pd5vz-zm5bm -- bash
root@anck-credentials-774f6cc648-pd5vz-zm5bm:/app# go run cmd/anck-credentials/main.go

# Start another shell with anck
$ kubectl exec -it anck-controller-manager-666cb6bf45-gztcs-2rr9m -c manager -- bash
root@anck-controller-manager-666cb6bf45-gztcs-2rr9m:/app# go run main.go
```

## Debugging while developing

Choose which component you want to debug and start it whith dlv.

```bash
$ kubectl ns anck
Active namespace is "anck".
$ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
anck-controller-manager-666cb6bf45-gztcs-2rr9m   2/2     Running   0          11h
anck-credentials-774f6cc648-pd5vz-zm5bm          1/1     Running   0          11h

# Start one shell with anck-credentials using dlv
$ kubectl exec -it anck-credentials-774f6cc648-pd5vz-zm5bm -- bash
root@anck-credentials-...:/app# dlv debug ./cmd/credsmanager/main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags="-mod=vendor"

# Start another shell with anck using dlv
$ kubectl exec -it anck-controller-manager-666cb6bf45-gztcs-2rr9m -c manager -- bash
root@anck-controller-manager-...:/app# dlv debug main.go --listen=0.0.0.0:2345 --api-version=2 --output /tmp/__debug_bin --headless --build-flags="-mod=vendor"
```

## Debugging notes for `anck-credentials`

If `anck` is not able to connect to `anck-credentials` using the service URI `anck-credentials.anck.svc.cluster.local`, this means that the service is not applicable for the `anck-credentials` pod. Please check if the labels selector for the service match the pods labels.
