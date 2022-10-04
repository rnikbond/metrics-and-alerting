FROM golang:latest

WORKDIR  /go/src/metrics-and-alerting
COPY . .
RUN go mod download
RUN go build -o /home/agent cmd/agent/main.go
RUN go build -o /home/server cmd/server/main.go

COPY start.sh /home/
RUN chmod a+x start.sh

EXPOSE 8080
WORKDIR /home
CMD ["/home/start.sh"]