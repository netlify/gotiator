FROM calavera/go-glide:v0.12.2

ADD . /go/src/github.com/netlify/gotiator

RUN useradd -m netlify && cd /go/src/github.com/netlify/gotiator && make deps build && mv gotiator /usr/local/bin/

USER netlify
CMD ["gotiator", "serve"]
