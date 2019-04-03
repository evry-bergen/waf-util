FROM alpine

WORKDIR /app
COPY main .

CMD ["/app/main"]