FROM alpine:3.6

ENV VERSION v2.7.2
ENV FILENAME helm-${VERSION}-linux-amd64.tar.gz

COPY chartmuseum.sh /usr/local/bin/chartmuseum.sh

RUN apk add --no-cache bash gawk sed grep bc coreutils curl openssl jq \
    && chmod +x /usr/local/bin/chartmuseum.sh \
    && curl -sLo /tmp/${FILENAME} http://storage.googleapis.com/kubernetes-helm/${FILENAME} \
    && tar -zxvf /tmp/${FILENAME} -C /tmp \
    && mv /tmp/linux-amd64/helm /bin/helm \
    && rm -rf /tmp 


ENTRYPOINT /usr/local/bin/chartmuseum.sh
