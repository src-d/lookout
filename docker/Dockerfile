FROM debian:stretch-slim

RUN apt-get update && \
    apt-get install --no-install-recommends -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY ./build/bin/lookoutd /bin/lookoutd

ENTRYPOINT ["/bin/lookoutd"]
CMD [ "serve" ]
