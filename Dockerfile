FROM alpine

WORKDIR /app
COPY main .
RUN apk --no-cache add ca-certificates

CMD ["./main"]