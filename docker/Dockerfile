FROM debian:stretch-slim

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ADD ./build/bin/lookout /bin/lookout

ENTRYPOINT ["/bin/lookout"]
CMD [ "serve" ]
