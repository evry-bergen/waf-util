FROM alpine

WORKDIR /app
COPY main .

CMD ["./main"]