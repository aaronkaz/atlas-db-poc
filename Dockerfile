FROM golang:1

# Create app directory
WORKDIR /opt

# Bring source into the container and build
COPY . .
RUN go mod download
RUN go build -o /opt/server

EXPOSE 9000

CMD [ "/opt/server" ]
