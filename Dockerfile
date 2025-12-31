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

FROM scratch

COPY --from=builder /app/github-review-helper /bin/github-review-helper

ENV PORT=80
EXPOSE ${PORT}

ENTRYPOINT [ "/bin/github-review-helper" ]
