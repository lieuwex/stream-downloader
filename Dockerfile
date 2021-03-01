FROM golang:1.15.5-alpine AS builder

# Install dep
RUN apk update && apk add git
RUN go get -u github.com/golang/dep/cmd/dep

# Install dependencies
COPY Gopkg.lock Gopkg.toml /go/src/stream-downloader/
WORKDIR /go/src/stream-downloader/
RUN dep ensure -vendor-only

COPY . .
RUN go build -o /bin/stream-downloader

#####

FROM python:3-alpine AS runner

RUN apk update \
	&& apk add --no-cache \
		ffmpeg \
		tzdata \
		gcc \
		musl-dev \
		bash \
		libvpx \
	&& pip install \
		streamlink \
		Av1an \
	&& apk del --purge --force \
		linux-headers \
		binutils-gold \
		gnupg \
		zlib-dev \
		libc-utils \
		gcc \
		musl-dev \
	&& rm -rf /var/lib/apt/lists/* \
		/var/cache/apk/* \
		/usr/share/man \
		/tmp/*

# Copy convert script
COPY ./convert/convert /bin/convert

# Copy stream-downloader
COPY --from=builder /bin/stream-downloader /bin/stream-downloader

WORKDIR /root
CMD ["/bin/stream-downloader", "/root/"]
