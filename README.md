# probegen

A small program to query the Backstage API and generate Cloudprober definitions based on annotation metadata.

The generated Cloudprober definitions can be used to monitor the health and availability of the components defined in your Backstage instance.

## Usage

See `./probegen -h` for options.

Given a backstage component such as:

```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: user-service
  description: User service
  tags:
    - aws
  annotations:
    backstage.io/techdocs-ref: dir:.
    example.com/probe-targets: user-service.prod.example.com
    example.com/probe-http-relative-url: /health
spec:
  type: service
  lifecycle: production
  owner: floobteam
  system: website
```

You can generate a probe definition like so:

```console
$ ./probegen -pretty -backstage-url 'https://backstage.ops.example.com' -namespace 'example.com' 2>/dev/null
probe: {
  name: "probe-user-service"
  type: HTTP
  interval: "10s"
  targets: {
    host_names: "user-service.prod.example.com"
  }
  http_probe: {
    protocol: HTTPS
    relative_url: "/health"
    method: GET
  }
}
```

## Contributing

Contributions to this project are welcome. If you encounter any issues or have suggestions for improvements, please open an issue or submit a pull request on the GitHub repository.

## Acknowledgments

This project is inspired by the [Backstage](https://backstage.io/) platform and [Cloudprober](https://cloudprober.org/). Special thanks to the contributors of these projects for their valuable work.
