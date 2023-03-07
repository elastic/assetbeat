FROM ubuntu:20.04

RUN mkdir -p /usr/share/inputrunner
COPY inputrunner /usr/share/inputrunner/inputrunner

RUN mkdir -p /usr/share/inputrunner/data /usr/share/inputrunner/logs && \
    chown -R root:root /usr/share/inputrunner && \
    find /usr/share/inputrunner -type d -exec chmod 0755 {} \; && \
    find /usr/share/inputrunner -type f -exec chmod 0644 {} \; && \
    chmod 0775 /usr/share/inputrunner/data /usr/share/inputrunner/logs


RUN chmod 0755 /usr/share/inputrunner/inputrunner
RUN for iter in {1..10}; do \
        apt-get update -y && \
        DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends --yes ca-certificates curl coreutils gawk libcap2-bin xz-utils && \
        apt-get clean all && \
        exit_code=0 && break || exit_code=$? && echo "apt-get error: retry $iter in 10s" && sleep 10; \
    done; \
    (exit $exit_code)


RUN groupadd --gid 1000 inputrunner
RUN useradd -M --uid 1000 --gid 1000 --groups 0 --home /usr/share/inputrunner inputrunner
USER inputrunner

WORKDIR /usr/share/inputrunner
CMD [ "/bin/bash", "-c", "./inputrunner", "run" ]