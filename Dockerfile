FROM golang:1.17-alpine

WORKDIR /app

# Put this before copying the full src over so we don't reinstall
# the required modules every time a file is changed.
RUN mkdir src
COPY server/go.mod go.mod
COPY server/go.sum go.sum
RUN go mod download

COPY server .
RUN go build -o ./webServer

COPY build build

EXPOSE 8080

CMD [ "./webServer" ]
