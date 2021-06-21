FROM golang:1.16-alpine

RUN mkdir /app

COPY . /app

WORKDIR /app

RUN go build -o server .

CMD [ "/app/server" ]
