# HAProxy GeoIP

This repo is a way to get GeoIP data into HAProxy using datasets from [IP Geolocation by DB-IP](https://db-ip.com). The
free datasets are downloaded monthly and a new container image is generated from that dataset. The data is converted
into a map file format that can be used directly inside HAProxy.

## Using the image

The container images are intended to be used as image volumes in Kubernetes, mounted alongside the HAProxy ingress
controller container. This requires Kubernetes 1.35 at a minimum and containerd 2.1.0 at a minimum.

```yaml
spec:
  volumes:
    - name: geoip
      image:
        reference: ghcr.io/monetr/haproxy-geoip:2026-06
        pullPolicy: IfNotPresent
  containers:
    - name: haproxy
      volumeMounts:
        - name: geoip
          mountPath: /etc/haproxy/geoip
```

Updating the GeoIP data is as simple as periodically bumping the image reference.

You can then configure HAProxy with a frontend snippet like this, which will load the country and the ASN for the client
IP address.

```haproxy
frontend fe_main
    bind :443 ssl crt /etc/haproxy/certs/

    # country (default XX for IPs not in the map)
    http-request set-var(txn.country) src,map_ip(/etc/haproxy/geoip/country.map,XX)

    # asn + org, packed as "ASN|org"
    http-request set-var(txn.asn)    src,map_ip(/etc/haproxy/geoip/asn.map),field(1,|)
    http-request set-var(txn.asorg)  src,map_ip(/etc/haproxy/geoip/asn.map),field(2,|)

    # example: block a country and forward the rest
    acl blocked var(txn.country) -m str KP
    http-request deny if blocked
    http-request set-header X-Country %[var(txn.country)]
    http-request set-header X-ASN     %[var(txn.asn)]
```
