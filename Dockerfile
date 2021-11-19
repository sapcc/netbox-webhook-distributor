FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/golang:1.17-alpine as builder
RUN apk add --no-cache make git curl

COPY . /src
RUN make -C /src install PREFIX=/pkg GO_BUILDFLAGS='-mod vendor'

FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/alpine:3.12
LABEL maintainer="Stefan Hipfel <stefan.hipfel@sap.com>"
LABEL source_repository="https://github.com/sapcc/netbox-webhook-distributor"

WORKDIR /
RUN apk --no-cache add curl
RUN curl -Lo /bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 \
	&& chmod +x /bin/dumb-init \
	&& dumb-init -V

COPY --from=builder /pkg/ /usr/
ENTRYPOINT ["dumb-init", "--"]
CMD ["/usr/bin/webhook"]
