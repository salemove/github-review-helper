FROM 662491802882.dkr.ecr.us-east-1.amazonaws.com/golang:1.23-20250306-b193

ENV PORT 80
EXPOSE $PORT

RUN echo "\
    UserKnownHostsFile /etc/secret-volume/known_hosts\n\
    IdentityFile /etc/secret-volume/id_rsa\n\
" >> /etc/ssh/ssh_config

ADD . /go/src/github.com/salemove/github-review-helper
RUN go get -v -d github.com/salemove/github-review-helper && \
  go install github.com/salemove/github-review-helper

ENTRYPOINT /go/bin/github-review-helper

