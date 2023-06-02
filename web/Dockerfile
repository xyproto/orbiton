FROM alpine
LABEL maintainer="xyproto@archlinux.org"
WORKDIR /app

# Copy in a HTML file, for now
COPY index.html index.html
COPY favicon.ico favicon.ico

# Create a minimal image for running the Algernon web server
FROM alpine
WORKDIR /srv

# Copy over the application that is about to be served
COPY --from=0 /app /srv/orbiton.zip

# Also add a symlink for localhost to one of the domains
RUN ln -s /srv/orbiton.zip /srv/localhost

# Install Algernon
ENV ALGERNON_VERSION='1.15.1'
ADD "https://github.com/xyproto/algernon/releases/download/v$ALGERNON_VERSION/algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" .
RUN tar Jxf "algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" && \
    install -Dm755 "algernon-$ALGERNON_VERSION-linux_x86_64_static/algernon" /usr/bin/algernon && \
    rm -r "algernon-$ALGERNON_VERSION-linux_x86_64_static" "algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" && \
    algernon --version

# Serve the current files on port 8080
EXPOSE 8080
EXPOSE 8080/udp
CMD /usr/bin/algernon -u --statcache --domain --server --cachesize 67108864 --cache=images /srv :8080
