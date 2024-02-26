FROM golang:1.21-alpine
RUN addgroup -S mercari && adduser -S trainee -G mercari
# RUN chown -R trainee:mercari /path/to/db
#dockerのパソコンapp/
WORKDIR /app/
#mercari.bild Mac何を環境dockerパスapp/db,app,images
COPY ./go /app/

RUN mv db /db && chown -R trainee:mercari /db

RUN go mod tidy
#docker環境ないのgo
CMD go run app/main.go

