FROM alpine
ADD cap-api-gateway /cap-api-gateway
ENTRYPOINT [ "/cap-api-gateway" ]
