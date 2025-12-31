ARG BUILDPLATFORM

FROM --platform=${BUILDPLATFORM} golang:1.25 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
ADD . /app

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

RUN make build

FROM alpine

COPY --from=builder /app/github-review-helper /bin/github-review-helper

RUN apk add --update --no-cache git openssh \
  && echo "UserKnownHostsFile /etc/secret-volume/known_hosts" >> /etc/ssh/ssh_config \
  && echo "IdentityFile /etc/secret-volume/id_rsa" >> /etc/ssh/ssh_config

ENV PORT=80
EXPOSE ${PORT}

ENTRYPOINT [ "/bin/github-review-helper" ]
