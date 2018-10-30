FROM golang:1.10.1-alpine3.7 AS builder

RUN apk update && apk add git
RUN go get -u github.com/golang/dep/cmd/dep

RUN mkdir -p /go/src/stream-downloader
WORKDIR /go/src/stream-downloader
COPY . .

RUN dep ensure
RUN GOOS=linux go build -o stream-downloader

#####

FROM python:3-alpine AS runner

# Update apk repositories
RUN apk update

# Install ffmpeg
RUN apk add ffmpeg --no-cache

# Install streamlink
RUN apk add gcc musl-dev --no-cache && pip install streamlink

# Copy stream-downloader and install deps
RUN apk add tzdata
COPY --from=builder /go/src/stream-downloader /bin/

# Remove unneeded stuff
RUN apk del --purge --force \
		linux-headers \
		binutils-gold \
		gnupg \
		zlib-dev \
		libc-utils \
		gcc \
		musl-dev \
	&& \
	rm -rf /var/lib/apt/lists/* \
		/var/cache/apk/* \
		/usr/share/man \
		/tmp/*

CMD ["/bin/stream-downloader", "/root/"]
