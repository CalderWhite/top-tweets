FROM golang:1.16-alpine

WORKDIR /app

# Put this before copying the full src over so we don't reinstall
# the required modules every time a file is changed.
RUN mkdir src
COPY src/go.mod src/go.mod
COPY src/go.sum src/go.sum
RUN cd src && go mod download

COPY src src
RUN cd src && go build -o ../webServer

COPY public public

EXPOSE 8080

CMD [ "./webServer" ]
