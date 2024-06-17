FROM alpine
LABEL maintainer="Alexader F. Rødseth <xyproto@archlinux.org>"
WORKDIR /app

# Copy in a HTML file, for now
COPY favicon.ico favicon.ico
COPY index.html index.html
COPY main.js main.js
COPY robots.txt robots.txt
COPY screenshot1.png screenshot1.png
COPY screenshot2.png screenshot2.png
COPY screenshot3.png screenshot3.png
COPY sitemap.xml sitemap.xml
COPY style.css style.css

# Create a minimal image for running the Algernon web server
FROM alpine
WORKDIR /srv

# Copy over the application that is about to be served
COPY --from=0 /app /srv/roboticoverlords.org/orbiton

# Also add a symlink for localhost to one of the domains
RUN ln -s /srv/roboticoverlords.org/orbiton /srv/localhost

# Install Algernon
ENV ALGERNON_VERSION='1.15.3'
ADD "https://github.com/xyproto/algernon/releases/download/v$ALGERNON_VERSION/algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" .
RUN tar Jxf "algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" && \
    install -Dm755 "algernon-$ALGERNON_VERSION-linux_x86_64_static/algernon" /usr/bin/algernon && \
    rm -r "algernon-$ALGERNON_VERSION-linux_x86_64_static" "algernon-$ALGERNON_VERSION-linux_x86_64_static.tar.xz" && \
    algernon --version

# Serve the current files on port 8080
EXPOSE 8080
EXPOSE 8080/udp
CMD /usr/bin/algernon -u --statcache --domain --server --cachesize 67108864 --cache=images /srv :8080
