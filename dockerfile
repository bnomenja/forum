
FROM golang:1.24-alpine


LABEL org.opencontainers.image.authors="obenmbar & mohnouri & wkhlifi bnomenja"

LABEL org.opencontainers.image.title="Forum"

LABEL org.opencontainers.image.description="A Go forum application using SQLite"

LABEL org.opencontainers.image.version="1.0.0"

LABEL org.opencontainers.image.licenses="MIT"

LABEL org.opencontainers.image.source="https://learn.zone01oujda.ma/git/wkhlifi/forum.git"


RUN apk add --no-cache gcc musl-dev


WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download


COPY . .


RUN CGO_ENABLED=1 go build -o main .

EXPOSE 8080

CMD ["./main"]