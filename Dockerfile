FROM golang:1.22
LABEL org.opencontainers.image.source="https://github.com/gemfast/server"
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ADD internal ./internal
ADD cmd ./cmd
ADD bin ./bin

COPY *.go ./
COPY Makefile ./

RUN mkdir -p /opt/gemfast/etc/gemfast && \
  mkdir -p /var/gemfast && \
  mkdir -p /etc/gemfast && \
  make

COPY gemfast_acl.csv /opt/gemfast/etc/gemfast/
COPY auth_model.conf /opt/gemfast/etc/gemfast/

EXPOSE 2020

CMD ["/app/bin/gemfast-server", "start"]