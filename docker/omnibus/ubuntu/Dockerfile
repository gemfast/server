ARG OS_IMAGE=ubuntu:jammy

FROM $OS_IMAGE

LABEL maintainer="Greg Schofield <greg@gemfast.io>"

ARG GO_VERSION=1.19.3

RUN apt update -y && \
DEBIAN_FRONTEND=noninteractive \
apt-get install -y curl gcc file fakeroot && \
curl -OL https://golang.org/dl/go$GO_VERSION.linux-arm64.tar.gz && \
tar -C /usr/local -xzf go$GO_VERSION.linux-arm64.tar.gz

RUN curl -fsSL https://omnitruck.chef.io/chef/install.sh | bash -s -- -c "stable" -P "omnibus-toolchain"

ENV PATH /opt/omnibus-toolchain/bin:/opt/omnibus-toolchain/embedded/bin:/usr/local/go/bin:${PATH}
RUN echo "${PATH}" >> /etc/bash.bashrc
CMD ["bash"]
