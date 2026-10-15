ARG BASE_IMAGE=ubuntu:24.04@sha256:534baea6a22c03a63003dbc8dbe78fe34bc0d7e595d9a9dc9834884ff530eb55
FROM ${BASE_IMAGE}
ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PATH=/opt/ptsip/bin:$PATH \
    HOME=/home/ptsip
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates python3 python3-venv \
    && rm -rf /var/lib/apt/lists/* \
    && python3 -m venv /opt/ptsip
COPY wheels/ /tmp/ptsip-wheels/
RUN python -m pip install --no-cache-dir --index-url https://pypi.org/simple /tmp/ptsip-wheels/*.whl \
    && rm -rf /tmp/ptsip-wheels
COPY shipping-manifest.json /opt/ptsip-release/shipping-manifest.json
RUN groupadd --gid 10001 ptsip \
    && useradd --uid 10001 --gid 10001 --create-home ptsip \
    && mkdir -p /workspace \
    && chown 10001:10001 /workspace
USER 10001:10001
WORKDIR /workspace
ENTRYPOINT ["ptsip"]
CMD ["--help"]
