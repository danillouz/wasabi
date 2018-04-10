# Wasabi

The webcontainers API.

It's Like Function as a Service (FaaS), but with 🐳. Instead of uploading a (NodeJS) function that
can be executed by calling a unique URL, you run an executable (one-off) Docker Image, by making a
HTTP POST request.

Wasabi ❤️ Kubernetes.

## Why?

Why not? 😁

## How?

You create configurations, where each configuration specifies a Docker Image, that will run to
completion each time the Wasabi API is called with the configuration name, where:

* A configuration is a [k8s secret](https://kubernetes.io/docs/concepts/configuration/secret/).
* The secret contains the Docker Image as a `base64` encoded data property.
* The secret name is used as the URL parameter when calling the Wasabi API, to execute the desired Image.
* The Docker Image muste be a one-off executable; it must run to completion.
* To start the process of running the container, a HTTP POST request must be made to the `/jobs/<SECRET_NAME>` endpoint of the Wasabi API.
* A JSON payload can be sent to provide Env Vars for the container.
* The entire JSON payload can be passed as an argument to the created container.
* A [k8s Job](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/) is created inside the cluster on which Wasabi is installed, everytime the Wasabi API is called.
* The container runs to completion, inside the Pod that was created by the Job.

The Wasabi API is a simple Go HTTP API that can be installed on any k8s cluster using [Helm](https://helm.sh/).

### 1. Install

First install the Wasabi API on your k8s cluster in a separate namespace by running:

```sh
> helm install ./charts/wasabi --namespace webcontainers --set wasabi.namespace=webcontainers --name wasabi
```

_It's recommended to install Wasabi in it's own namespace._

This will create a service of type `LoadBalancer`:

```sh
> kub get service

NAME      TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)        AGE
wasabi    LoadBalancer   10.63.241.41   34.196.174.132   80:31878/TCP   18m
```

Use the external IP and send a GET request to the `/health` endpoint, to see if Wasabi can receive
traffic:

```sh
> curl http://34.196.174.132/health

{"message":"OK"}
```

### 2. Configure

Creating a config, means creating a k8s secret, which must contain `bas64` encoded data values. We
have to provide the Docker Image we wish to execute, for example [danillouz/docker-say](https://hub.docker.com/r/danillouz/docker-say/). This is a simple NodeJS script that prints passed
arguments to STDOUT, just for demo purposes. The source code can be viewed [here](https://github.com/danillouz/docker-say).

Encode the Docker Image name:

```sh
> echo -n "danillouz/docker-say" | base64

ZGFuaWxsb3V6L2RvY2tlci1zYXk=
```

Create the k8s manifest file:

```sh
> cat <<EOF > docker-say.yaml
apiVersion: v1
kind: Secret
metadata:
  name: webcontainers-config-docker-say
  namespace: webcontainers
type: Opaque
data:
  image: ZGFuaWxsb3V6L2RvY2tlci1zYXk=
EOF
```

Create the k8s secret:

```sh
> kubectl create -f docker-say.yaml -n webcontainers

secret "webcontainers-config-docker-say" created
```

### 3. Execute

Now we can use the secret name `webcontainers-config-docker-say`, to execute the configured Docker
Image by sending a HTTP POST request to the `jobs/webcontainers-config-docker-say` endpoint of the
Wasabi API:

```sh
curl -H "Content-Type: application/json" -X POST -d '{"data":"Hello World!"}' http://34.196.174.132/jobs/webcontainers-config-docker-say

{"UID":"990ce026-3cfb-11e8-bf98-42010a840020","message":"OK","name":"webcontainer-1523391193"}
```

This created Job `webcontainer-1523391193` and if we inspect the logs of the container that was run,
we'll see:

```sh
> kubectl logs jobs/webcontainer-1523391193 -n webcontainers

say.js args:  [ '{"config":{"image":"danillouz/docker-say"},"payload":{"data":"Hello World!"}}' ]
```

This single JSON string contains a `config` property, which contains all `data` properties specified
in the corresponding secret. And a `payload` property, which contains the entire JSON payload that
was sent when making the HTTP POST request. The NodeJS script can print it by accessing `process.argv`.

#### Another Example

Some executable Docker Images don't accept arguments but are configured purely by passing Env Vars,
for example [slack-notify](https://github.com/technosophos/slack-notify). We can still execute Images
like this with Wasabi by sending the following JSON payload:

```json
{
  "containerEnvVars": [
    {
      "name": "SLACK_WEBHOOK",
      "value": "https://hooks.slack.com/services/xxxxxxxxx/xxxxxxxxx/xxxxxxxxxxxxxxxxxxxxxxxx"
    },
    { "name": "SLACK_TITLE", "value": "Wasabi is alive!" },
    { "name": "SLACK_MESSAGE", "value": "Hello World!" },
    { "name": "SLACK_COLOR", "value": "#d1f1a9" }
  ],
  "passContainerArg": false
}
```

By providing the property `passContainerArg` with value `false`, the JSON payload is NOT passed as
an argument.

## Disclaimer

At this stage Wasabi is just a proof of concept and I advise not to run it in production, yet. But
it's awesome for experimentation and executing webhook.
