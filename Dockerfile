FROM golang:alpine as builder
# Below ENV should be turned on when it is not possible to access google
# ENV GO111MODULE=on
# ENV GOPROXY=https://goproxy.io
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o portops
FROM scratch
COPY --from=builder /build/portops /app/
WORKDIR /app
CMD ["./portops"]
