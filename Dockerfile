FROM calavera/go-glide:v0.12.2

ADD . /go/src/github.com/netlify/netlify-api-proxy

RUN useradd -m netlify && cd /go/src/github.com/netlify/netlify-api-proxy && make deps build && mv netlify-api-proxy /usr/local/bin/

USER netlify
CMD ["netlify-api-proxy"]
