FROM golang:1.21-alpine
RUN addgroup -S mercari && adduser -S trainee -G mercari

#dockerのパソコンapp/
WORKDIR /app/
#mercari.bild 第一引数：Mac環境の何を、第二引数：docker環境のapp/配下に置くかを指定！
COPY ./go /app/

RUN mv db /db && chown -R trainee:mercari /db

RUN go mod tidy
#docker環境内sのgo
CMD go run app/main.go

