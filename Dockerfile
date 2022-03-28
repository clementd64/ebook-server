FROM golang:1.18 as builder

WORKDIR /app
COPY . .
RUN go build -o main .

FROM python:3.10-bullseye

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends libgl1 xz-utils \
    && wget -nv -O- https://download.calibre-ebook.com/linux-installer.sh | sh /dev/stdin \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main /main
ENTRYPOINT [ "/main" ]