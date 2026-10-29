FROM --platform=$BUILDPLATFORM golang:1.27 AS build
WORKDIR /build
COPY . ./
RUN go build -o /geoip2map .

FROM --platform=$BUILDPLATFORM debian:13 AS data
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl gzip && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /geoip2map /usr/local/bin/geoip2map
WORKDIR /work
# Defaults to the current UTC year and month (YYYY-MM) when not provided.
ARG DBIP_DATE
RUN set -eux; \
    DBIP_DATE="${DBIP_DATE:-$(date -u +%Y-%m)}"; \
    mkdir -p /out; \
    base="https://download.db-ip.com/free"; \
    # DB-IP Lite: CC BY 4.0, monthly release, no account.
    curl -fSL -o country.mmdb.gz "${base}/dbip-country-lite-${DBIP_DATE}.mmdb.gz"; \
    curl -fSL -o asn.mmdb.gz     "${base}/dbip-asn-lite-${DBIP_DATE}.mmdb.gz"; \
    gunzip country.mmdb.gz; \
    gunzip asn.mmdb.gz; \
    geoip2map -db country.mmdb -mode country -out /out/country.map; \
    geoip2map -db asn.mmdb     -mode asn     -out /out/asn.map; \
    printf 'source=db-ip.com lite\ndate=%s\n' "$DBIP_DATE" > /out/VERSION; \
    wc -l /out/*.map

FROM scratch
COPY --from=data /out/ /
