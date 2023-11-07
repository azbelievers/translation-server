
FROM node:18-alpine as clientbuilder

WORKDIR /app

COPY ./client/package.json ./client/package-lock.json ./

RUN npm install

COPY ./client/ ./

RUN npm run build

####

FROM golang:1.21.3-alpine as serverbuilder

WORKDIR /app

COPY ./server/go.mod ./server/go.sum ./

RUN go mod download

COPY ./server/ ./

RUN go build -o main .

####

FROM alpine:latest as production

WORKDIR /app

COPY --from=clientbuilder /app/dist ./client
COPY --from=serverbuilder /app/main ./

#ENV GIN_MODE=release

EXPOSE 8080

CMD ["./main"]