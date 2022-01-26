FROM golang:1.17-alpine

WORKDIR /app

# Put this before copying the full src over so we don't reinstall
# the required modules every time a file is changed.
RUN mkdir src
RUN mkdir backups
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY web/build build


COPY lib lib
COPY *.go ./
RUN go build -o ./webServer top_tweets.go twitter_worker.go

EXPOSE 8080

CMD [ "./webServer" ]
