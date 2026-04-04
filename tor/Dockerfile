FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends tor ca-certificates && rm -rf /var/lib/apt/lists/*
COPY snowflake-client /usr/bin/snowflake-client
COPY torrc /etc/tor/torrc
USER debian-tor
EXPOSE 9050
CMD ["tor", "-f", "/etc/tor/torrc"]
